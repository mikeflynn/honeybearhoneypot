package geo

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"

	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/geo/data"
	maxminddb "github.com/oschwald/maxminddb-golang/v2"
)

type Geo struct {
	IP          string
	Lat         float64
	Lon         float64
	Country     string // human-readable name, e.g. "United States"
	CountryCode string // ISO 3166-1 alpha-2, e.g. "US"
}

var (
	reader   *maxminddb.Reader
	readerMu sync.RWMutex
)

// mmdbRecord matches the Country-DB schema fields we care about.
type mmdbRecord struct {
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
}

// Init opens the embedded mmdb. Safe to call multiple times.
func Init() error {
	readerMu.Lock()
	defer readerMu.Unlock()
	if reader != nil {
		return nil
	}
	r, err := maxminddb.OpenBytes(data.MMDB)
	if err != nil {
		return fmt.Errorf("geo: open mmdb: %w", err)
	}
	reader = r
	return nil
}

// Close releases the reader. For tests / shutdown.
func Close() {
	readerMu.Lock()
	defer readerMu.Unlock()
	if reader != nil {
		_ = reader.Close()
		reader = nil
	}
}

// Lookup returns geo info for ip. Returns (nil, nil) for private/loopback IPs
// or for countries with no centroid entry (so callers can skip them).
// Cache-through: hits geo_cache first, falls back to the mmdb on cache miss
// and writes the result back.
func Lookup(ip string) (*Geo, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return nil, fmt.Errorf("invalid ip: %q", ip)
	}
	if isPrivate(parsed) {
		return nil, nil
	}

	if cached, err := entity.GeoCacheGet(ip); err == nil && cached != nil {
		return &Geo{
			IP:          cached.IP,
			Lat:         cached.Lat,
			Lon:         cached.Lon,
			Country:     cached.Country,
			CountryCode: cached.CountryCode,
		}, nil
	}

	readerMu.RLock()
	r := reader
	readerMu.RUnlock()
	if r == nil {
		return nil, errors.New("geo: not initialized; call Init() first")
	}

	addr, ok := netip.AddrFromSlice(parsed)
	if !ok {
		return nil, fmt.Errorf("geo: could not convert IP %s to netip.Addr", ip)
	}
	addr = addr.Unmap()

	var rec mmdbRecord
	if err := r.Lookup(addr).Decode(&rec); err != nil {
		return nil, fmt.Errorf("geo: lookup %s: %w", ip, err)
	}
	iso := rec.Country.ISOCode
	if iso == "" {
		return nil, nil
	}
	centroid, ok := Centroids[iso]
	if !ok {
		return nil, nil
	}

	g := &Geo{
		IP:          ip,
		Lat:         centroid[0],
		Lon:         centroid[1],
		Country:     pickName(rec.Country.Names, iso),
		CountryCode: iso,
	}
	_ = entity.GeoCacheUpsert(&entity.GeoCache{
		IP:          g.IP,
		Lat:         g.Lat,
		Lon:         g.Lon,
		Country:     g.Country,
		CountryCode: g.CountryCode,
	})
	return g, nil
}

func pickName(names map[string]string, fallback string) string {
	if v, ok := names["en"]; ok && v != "" {
		return v
	}
	return fallback
}

func isPrivate(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, cidr := range []string{
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"100.64.0.0/10", "fc00::/7",
	} {
		_, n, _ := net.ParseCIDR(cidr)
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

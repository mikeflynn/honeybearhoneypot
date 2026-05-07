package entity

import (
	"database/sql"
	"errors"
	"time"

	"github.com/mikeflynn/honeybearhoneypot/internal/db"
)

func GeoCacheInitialization() string {
	return `
		CREATE TABLE IF NOT EXISTS geo_cache (
			ip TEXT PRIMARY KEY,
			lat REAL NOT NULL,
			lon REAL NOT NULL,
			country TEXT NOT NULL,
			country_code TEXT NOT NULL,
			looked_up_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`
}

type GeoCache struct {
	IP          string
	Lat         float64
	Lon         float64
	Country     string
	CountryCode string
	LookedUpAt  time.Time
}

// GeoCacheGet returns nil, nil when the IP is not cached.
func GeoCacheGet(ip string) (*GeoCache, error) {
	rows, err := db.MakeQuery(
		`SELECT ip, lat, lon, country, country_code, looked_up_at FROM geo_cache WHERE ip = ? LIMIT 1`,
		ip,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	g := &GeoCache{}
	if err := rows.Scan(&g.IP, &g.Lat, &g.Lon, &g.Country, &g.CountryCode, &g.LookedUpAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return g, nil
}

func GeoCacheUpsert(g *GeoCache) error {
	return db.MakeWrite(
		`INSERT INTO geo_cache (ip, lat, lon, country, country_code)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(ip) DO UPDATE SET
			 lat=excluded.lat,
			 lon=excluded.lon,
			 country=excluded.country,
			 country_code=excluded.country_code,
			 looked_up_at=CURRENT_TIMESTAMP`,
		g.IP, g.Lat, g.Lon, g.Country, g.CountryCode,
	)
}

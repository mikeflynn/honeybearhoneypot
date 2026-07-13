<?php
/**
 * Honey Bear Honey Pot — CTF leaderboard webhook receiver.
 *
 * Deploy at a hard-to-guess URL (the URL itself is the shared secret, matching
 * the honeypot's design). The website reads $STATE_FILE separately to render
 * the leaderboard.
 *
 * Suggested deployment:
 *   1. Copy this file somewhere web-accessible, e.g.
 *      /var/www/honeybear/wh-<random-32-chars>.php
 *   2. Create the state directory OUTSIDE the webroot and make it writable by
 *      the PHP-FPM user only:
 *        mkdir -p /var/lib/honeybear
 *        chown www-data:www-data /var/lib/honeybear
 *        chmod 700 /var/lib/honeybear
 *   3. Point the honeypot at:
 *        https://your.host/wh-<random-32-chars>.php
 *   4. Have the site read /var/lib/honeybear/leaderboard.json.
 */

declare(strict_types=1);

// ---- Config -----------------------------------------------------------------

$STATE_DIR   = '/var/lib/honeybear';           // outside webroot
$STATE_FILE  = $STATE_DIR . '/leaderboard.json';
$MAX_BYTES   = 64 * 1024;                       // reject payloads larger than 64KB
$FILE_MODE   = 0640;                            // rw for owner, r for group

// Optional second-factor: require this header to match. Leave null to rely on
// URL secrecy alone. If set, configure the honeypot's reverse proxy or a
// custom transport to send this header.
$SHARED_TOKEN = null; // e.g. 'change-me-to-a-long-random-string'
$TOKEN_HEADER = 'X-Honeybear-Token';

// ---- Helpers ----------------------------------------------------------------

function fail(int $code, string $msg): void {
    http_response_code($code);
    header('Content-Type: text/plain; charset=utf-8');
    echo $msg . "\n";
    exit;
}

function ok(): void {
    http_response_code(204);
    exit;
}

// ---- Request validation -----------------------------------------------------

if (($_SERVER['REQUEST_METHOD'] ?? '') !== 'POST') {
    header('Allow: POST');
    fail(405, 'method not allowed');
}

$contentType = strtolower(trim(explode(';', $_SERVER['CONTENT_TYPE'] ?? '')[0]));
if ($contentType !== 'application/json') {
    fail(415, 'expected application/json');
}

if ($SHARED_TOKEN !== null) {
    $headerKey = 'HTTP_' . str_replace('-', '_', strtoupper($TOKEN_HEADER));
    $provided  = $_SERVER[$headerKey] ?? '';
    if (!hash_equals($SHARED_TOKEN, (string) $provided)) {
        fail(401, 'unauthorized');
    }
}

$contentLength = (int) ($_SERVER['CONTENT_LENGTH'] ?? 0);
if ($contentLength > 0 && $contentLength > $MAX_BYTES) {
    fail(413, 'payload too large');
}

$raw = file_get_contents('php://input', false, null, 0, $MAX_BYTES + 1);
if ($raw === false) {
    fail(400, 'could not read body');
}
if (strlen($raw) > $MAX_BYTES) {
    fail(413, 'payload too large');
}

try {
    $data = json_decode($raw, true, 8, JSON_THROW_ON_ERROR);
} catch (JsonException $e) {
    fail(400, 'invalid json');
}

if (!is_array($data)
    || !isset($data['event'], $data['timestamp'], $data['source'], $data['leaderboard'])
    || !is_string($data['event'])
    || !is_string($data['timestamp'])
    || !is_string($data['source'])
    || !is_array($data['leaderboard'])
) {
    fail(400, 'unexpected payload shape');
}

// Whitelist known events to keep the state file predictable.
if (!in_array($data['event'], ['solve', 'startup', 'heartbeat'], true)) {
    fail(400, 'unknown event');
}

// Normalize + validate each leaderboard row.
$rows = [];
foreach ($data['leaderboard'] as $entry) {
    if (!is_array($entry)
        || !isset($entry['rank'], $entry['username'], $entry['points'])
        || !is_int($entry['rank'])
        || !is_string($entry['username'])
        || !is_int($entry['points'])
    ) {
        fail(400, 'invalid leaderboard row');
    }
    $rows[] = [
        'rank'     => $entry['rank'],
        'username' => $entry['username'],
        'points'   => $entry['points'],
    ];
}

$clean = [
    'event'         => $data['event'],
    'timestamp'     => $data['timestamp'],
    'source'        => $data['source'],
    'leaderboard'   => $rows,
    'received_at'   => gmdate('c'),
];

// ---- Atomic write -----------------------------------------------------------

if (!is_dir($STATE_DIR)) {
    fail(500, 'state dir missing');
}

$tmp = tempnam($STATE_DIR, '.lb-');
if ($tmp === false) {
    fail(500, 'could not create temp file');
}

$encoded = json_encode($clean, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE);
if ($encoded === false) {
    @unlink($tmp);
    fail(500, 'encode failed');
}

if (file_put_contents($tmp, $encoded, LOCK_EX) === false) {
    @unlink($tmp);
    fail(500, 'write failed');
}

@chmod($tmp, $FILE_MODE);

if (!rename($tmp, $STATE_FILE)) {
    @unlink($tmp);
    fail(500, 'rename failed');
}

ok();

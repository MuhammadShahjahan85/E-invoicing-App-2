<?php
/**
 * Tiny file-based storage for enquiries, subscribers and applications.
 * Files live in /storage and start with "<?php exit; ?>" so they can never be
 * read through the web even if the server ignored storage/.htaccess.
 */

const STORE_GUARD = "<?php exit; ?>\n";

function store_path(string $name): string
{
    return storage_dir() . '/' . preg_replace('/[^a-z0-9-]/', '', $name) . '.php';
}

/** Read all records of a store. */
function store_all(string $name): array
{
    $file = store_path($name);
    if (!is_file($file)) {
        return [];
    }
    $raw = (string) file_get_contents($file);
    if (strpos($raw, STORE_GUARD) === 0) {
        $raw = substr($raw, strlen(STORE_GUARD));
    }
    $data = json_decode($raw, true);
    return is_array($data) ? $data : [];
}

/** Change a store under an exclusive lock. $fn receives the records and returns them. */
function store_update(string $name, callable $fn): bool
{
    $file = store_path($name);
    $fh = @fopen($file, 'c+');
    if (!$fh) {
        return false;
    }
    try {
        if (!flock($fh, LOCK_EX)) {
            return false;
        }
        $raw = stream_get_contents($fh);
        if (strpos($raw, STORE_GUARD) === 0) {
            $raw = substr($raw, strlen(STORE_GUARD));
        }
        $records = json_decode($raw ?: '[]', true);
        if (!is_array($records)) {
            $records = [];
        }
        $records = $fn($records);
        $json = json_encode(array_values($records), JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES | JSON_PRETTY_PRINT);
        ftruncate($fh, 0);
        rewind($fh);
        fwrite($fh, STORE_GUARD . $json);
        fflush($fh);
        flock($fh, LOCK_UN);
        return true;
    } finally {
        fclose($fh);
    }
}

function store_add(string $name, array $record): bool
{
    return store_update($name, function (array $records) use ($record) {
        array_unshift($records, $record);
        return $records;
    });
}

function new_record_id(): string
{
    return date('ymd') . '-' . bin2hex(random_bytes(4));
}

/** Allow at most $max events per $window seconds for a key (e.g. form + IP). */
function rate_limited(string $key, int $max, int $window): bool
{
    $dir = storage_dir('ratelimit');
    $file = $dir . '/' . hash('sha256', $key . vp_secret()) . '.json';
    $now = time();
    $hits = [];
    if (is_file($file)) {
        $hits = json_decode((string) @file_get_contents($file), true) ?: [];
    }
    $hits = array_values(array_filter($hits, fn($t) => $t > $now - $window));
    if (count($hits) >= $max) {
        return true;
    }
    $hits[] = $now;
    @file_put_contents($file, json_encode($hits), LOCK_EX);
    // Occasionally clear old files.
    if (mt_rand(1, 50) === 1) {
        foreach (glob($dir . '/*.json') ?: [] as $f) {
            if (filemtime($f) < $now - 86400) {
                @unlink($f);
            }
        }
    }
    return false;
}

function client_ip(): string
{
    return substr((string) ($_SERVER['REMOTE_ADDR'] ?? ''), 0, 45);
}

function log_line(string $file, string $line): void
{
    $path = storage_dir() . '/' . $file . '.php';
    if (!is_file($path)) {
        @file_put_contents($path, STORE_GUARD, LOCK_EX);
    }
    @file_put_contents($path, '[' . date('Y-m-d H:i:s') . '] ' . str_replace(["\r", "\n"], ' ', $line) . "\n", FILE_APPEND | LOCK_EX);
}

/** Signed token for unsubscribe links. */
function unsubscribe_token(string $email): string
{
    return substr(hash_hmac('sha256', 'unsub|' . strtolower($email), vp_secret()), 0, 32);
}

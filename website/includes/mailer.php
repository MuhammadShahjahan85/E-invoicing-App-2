<?php
/**
 * E-mail sending: SMTP (recommended, settings in config.php) or PHP mail().
 */

function mail_from_address(): string
{
    $from = trim((string) cfg('forms.from_email', ''));
    if ($from === '' && cfg('smtp.enabled') && filter_var(cfg('smtp.username'), FILTER_VALIDATE_EMAIL)) {
        $from = (string) cfg('smtp.username');
    }
    return $from !== '' ? $from : 'info@' . site_domain();
}

function mime_header(string $text): string
{
    return preg_match('/[^\x20-\x7E]/', $text) ? '=?UTF-8?B?' . base64_encode($text) . '?=' : $text;
}

function clean_header_value(string $v): string
{
    return trim(str_replace(["\r", "\n", "\0"], '', $v));
}

/**
 * Send an HTML e-mail.
 * $to: list of addresses. $opts: reply_to, attachments => [[path, filename, mime]].
 */
function send_mail(array $to, string $subject, string $html, array $opts = []): bool
{
    $to = array_values(array_filter(array_map('trim', $to), fn($a) => filter_var($a, FILTER_VALIDATE_EMAIL)));
    if (!$to) {
        return false;
    }
    $from = clean_header_value(mail_from_address());
    $fromName = clean_header_value((string) cfg('forms.from_name', company_name()));
    $subject = clean_header_value($subject);
    $replyTo = isset($opts['reply_to']) && filter_var($opts['reply_to'], FILTER_VALIDATE_EMAIL) ? clean_header_value($opts['reply_to']) : '';

    $text = trim(html_entity_decode(strip_tags(preg_replace(['/<br\s*\/?>/i', '/<\/(p|tr|h\d|li|div)>/i'], "\n", $html)), ENT_QUOTES, 'UTF-8'));
    $alt = 'alt-' . bin2hex(random_bytes(8));
    $altBody = "--$alt\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n" . chunk_split(base64_encode($text))
        . "--$alt\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n" . chunk_split(base64_encode($html))
        . "--$alt--\r\n";

    $attachments = $opts['attachments'] ?? [];
    if ($attachments) {
        $mix = 'mix-' . bin2hex(random_bytes(8));
        $contentType = "multipart/mixed; boundary=\"$mix\"";
        $body = "--$mix\r\nContent-Type: multipart/alternative; boundary=\"$alt\"\r\n\r\n" . $altBody;
        foreach ($attachments as [$path, $name, $mime]) {
            if (!is_file($path)) {
                continue;
            }
            $safeName = preg_replace('/[^A-Za-z0-9._-]/', '_', $name);
            $body .= "--$mix\r\nContent-Type: $mime; name=\"$safeName\"\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: attachment; filename=\"$safeName\"\r\n\r\n"
                . chunk_split(base64_encode((string) file_get_contents($path)));
        }
        $body .= "--$mix--\r\n";
    } else {
        $contentType = "multipart/alternative; boundary=\"$alt\"";
        $body = $altBody;
    }

    $headers = [
        'From: ' . mime_header($fromName) . ' <' . $from . '>',
        'MIME-Version: 1.0',
        'Content-Type: ' . $contentType,
        'Date: ' . date('r'),
        'Message-ID: <' . bin2hex(random_bytes(12)) . '@' . site_domain() . '>',
        'X-Auto-Response-Suppress: OOF, AutoReply',
    ];
    if ($replyTo !== '') {
        $headers[] = 'Reply-To: ' . $replyTo;
    }

    if (cfg('smtp.enabled')) {
        $ok = smtp_send($from, $to, $subject, $headers, $body);
    } else {
        $headerStr = implode("\r\n", $headers);
        $subj = mime_header($subject);
        $ok = @mail(implode(', ', $to), $subj, $body, $headerStr, '-f' . $from);
        if (!$ok) {
            $ok = @mail(implode(', ', $to), $subj, $body, $headerStr);
        }
        if (!$ok) {
            log_line('mail-log', 'mail() failed for ' . implode(',', $to) . ': ' . $subject);
        }
    }
    return $ok;
}

/** Minimal SMTP client (SSL on 465 or STARTTLS on 587) with AUTH LOGIN. */
function smtp_send(string $from, array $to, string $subject, array $headers, string $body): bool
{
    $host = (string) cfg('smtp.host');
    $port = (int) cfg('smtp.port', 465);
    $secure = strtolower((string) cfg('smtp.secure', 'ssl'));
    $user = (string) cfg('smtp.username');
    $pass = (string) cfg('smtp.password');
    $remote = ($secure === 'ssl' ? 'ssl://' : 'tcp://') . $host . ':' . $port;
    $ctx = stream_context_create(['ssl' => ['verify_peer' => true, 'verify_peer_name' => true, 'SNI_enabled' => true]]);
    $fp = @stream_socket_client($remote, $errno, $errstr, 15, STREAM_CLIENT_CONNECT, $ctx);
    if (!$fp) {
        log_line('mail-log', "SMTP connect to $remote failed: $errstr ($errno)");
        return false;
    }
    stream_set_timeout($fp, 20);
    $read = function () use ($fp): string {
        $data = '';
        while (($line = fgets($fp, 1024)) !== false) {
            $data .= $line;
            if (strlen($line) < 4 || $line[3] !== '-') {
                break;
            }
        }
        return $data;
    };
    $cmd = function (string $c, array $expect, bool $mask = false) use ($fp, $read): bool {
        if ($c !== '') {
            fwrite($fp, $c . "\r\n");
        }
        $resp = $read();
        $code = (int) substr($resp, 0, 3);
        if (!in_array($code, $expect, true)) {
            $shown = $mask ? '[credentials]' : $c;
            log_line('mail-log', "SMTP '$shown' → " . trim($resp));
            return false;
        }
        return true;
    };
    $ehlo = 'EHLO ' . site_domain();
    $ok = $cmd('', [220]) && $cmd($ehlo, [250]);
    if ($ok && $secure === 'tls') {
        $ok = $cmd('STARTTLS', [220])
            && stream_socket_enable_crypto($fp, true, STREAM_CRYPTO_METHOD_TLSv1_2_CLIENT | STREAM_CRYPTO_METHOD_TLSv1_3_CLIENT)
            && $cmd($ehlo, [250]);
    }
    if ($ok && $user !== '') {
        $ok = $cmd('AUTH LOGIN', [334]) && $cmd(base64_encode($user), [334], true) && $cmd(base64_encode($pass), [235], true);
    }
    $ok = $ok && $cmd('MAIL FROM:<' . $from . '>', [250]);
    foreach ($to as $rcpt) {
        $ok = $ok && $cmd('RCPT TO:<' . $rcpt . '>', [250, 251]);
    }
    if ($ok && $cmd('DATA', [354])) {
        $msg = implode("\r\n", array_merge(['To: ' . implode(', ', $to), 'Subject: ' . mime_header($subject)], $headers)) . "\r\n\r\n" . $body;
        $msg = preg_replace('/^\./m', '..', str_replace(["\r\n", "\n"], ["\n", "\r\n"], $msg));
        fwrite($fp, $msg . "\r\n.\r\n");
        $ok = $cmd('', [250]);
    } else {
        $ok = false;
    }
    @fwrite($fp, "QUIT\r\n");
    fclose($fp);
    return $ok;
}

/** Branded HTML e-mail wrapper. */
function email_layout(string $title, string $content): string
{
    $brand = e(company_name());
    $site = e(abs_url());
    $legal = e(legal_name());
    return '<!doctype html><html><body style="margin:0;background:#f4f2ea;font-family:Segoe UI,Helvetica,Arial,sans-serif;color:#0e2a24">'
        . '<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f4f2ea;padding:24px 12px"><tr><td align="center">'
        . '<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%;background:#ffffff;border-radius:14px;overflow:hidden;border:1px solid #e6e3d8">'
        . '<tr><td style="background:#06281f;padding:22px 28px"><span style="font-family:Georgia,serif;font-size:22px;color:#ffffff;font-weight:bold">Veridian</span> <span style="font-size:11px;letter-spacing:3px;color:#e3b341;font-weight:bold">PARTNERS</span></td></tr>'
        . '<tr><td style="padding:28px"><h1 style="font-family:Georgia,serif;font-weight:normal;font-size:24px;margin:0 0 16px;color:#0e2a24">' . e($title) . '</h1>'
        . $content . '</td></tr>'
        . '<tr><td style="padding:18px 28px;background:#f7f6f1;font-size:12px;color:#667c76;border-top:1px solid #e6e3d8">' . $legal . ' · <a href="' . $site . '" style="color:#156b57">' . $site . '</a></td></tr>'
        . '</table></td></tr></table></body></html>';
}

/** Two-column table of submitted fields for notification e-mails. */
function email_fields_table(array $fields): string
{
    $rows = '';
    foreach ($fields as $label => $value) {
        if ($value === '' || $value === null || $value === []) {
            continue;
        }
        $value = is_array($value) ? implode(', ', $value) : (string) $value;
        $rows .= '<tr><td style="padding:8px 10px;border-bottom:1px solid #eee;color:#667c76;font-size:13px;width:34%;vertical-align:top">' . e($label) . '</td>'
            . '<td style="padding:8px 10px;border-bottom:1px solid #eee;font-size:14px;white-space:pre-wrap">' . e($value) . '</td></tr>';
    }
    return '<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="border-collapse:collapse;margin:8px 0 16px">' . $rows . '</table>';
}

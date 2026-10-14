<?php
/**
 * Receives every public form (contact, consultation, newsletter, careers):
 * spam checks → validation → saved to /storage → e-mailed to you → reply.
 */
require __DIR__ . '/includes/bootstrap.php';
require __DIR__ . '/includes/store.php';
require __DIR__ . '/includes/mailer.php';

$isAjax = (($_SERVER['HTTP_X_REQUESTED_WITH'] ?? '') === 'fetch') || strpos((string) ($_SERVER['HTTP_ACCEPT'] ?? ''), 'application/json') !== false;

function respond(bool $ok, string $message, array $extra = []): void
{
    global $isAjax, $form;
    if ($isAjax) {
        header('Content-Type: application/json; charset=utf-8');
        header('Cache-Control: no-store');
        if (!$ok) {
            http_response_code(isset($extra['errors']) ? 422 : 400);
        }
        echo json_encode(['ok' => $ok, 'message' => $message] + $extra, JSON_UNESCAPED_UNICODE);
    } else {
        header('Location: ' . url('thank-you?type=' . ($ok ? ($form ?: 'contact') : 'error')), true, 303);
    }
    exit;
}

function field(string $name, int $max = 500): string
{
    $v = $_POST[$name] ?? '';
    if (is_array($v)) {
        return '';
    }
    $v = trim(str_replace("\0", '', (string) $v));
    return mb_substr($v, 0, $max);
}

if (($_SERVER['REQUEST_METHOD'] ?? 'GET') !== 'POST') {
    header('Location: ' . url(), true, 302);
    exit;
}

$form = field('form', 20);
$allowed = ['contact', 'consultation', 'newsletter', 'career'];
if (!in_array($form, $allowed, true)) {
    respond(false, 'Unknown form.');
}

// --- Spam protection ----------------------------------------------------------
if (field('website') !== '') {
    // Bots fill the hidden field. Pretend all is well and discard.
    respond(true, 'Thank you.', ['title' => 'Thank you.']);
}
$tokenProblem = check_form_token(field('_t', 200), $form === 'newsletter' ? 2 : 3);
if ($tokenProblem === 'too-fast') {
    respond(false, 'That was very quick! Please wait a moment and send again.');
}
if ($tokenProblem !== '') {
    respond(false, 'This page has been open for a long time. Please refresh the page and send again.');
}
if (rate_limited($form . '|' . client_ip(), $form === 'newsletter' ? 5 : 6, 600)) {
    respond(false, 'We have received several messages from you in a short time. Please wait a few minutes, or e-mail us at ' . contact_email() . '.');
}

// --- Validation ----------------------------------------------------------------
$errors = [];
$email = field('email', 160);
if (!filter_var($email, FILTER_VALIDATE_EMAIL)) {
    $errors['email'] = 'Please enter a valid e-mail address.';
}
$name = field('name', 120);
$phone = field('phone', 40);
$message = field('message', 5000);

if ($form !== 'newsletter') {
    if (mb_strlen($name) < 2) {
        $errors['name'] = 'Please enter your name.';
    }
    if (field('consent') !== 'yes') {
        $errors['consent'] = 'Please tick this box to continue.';
    }
}
if ($phone !== '' && strlen(preg_replace('/\D/', '', $phone)) < 7) {
    $errors['phone'] = 'Please enter a valid phone number.';
}
if (in_array($form, ['consultation', 'career'], true) && $phone === '') {
    $errors['phone'] = 'Please enter a phone number.';
}
if ($form === 'contact' && mb_strlen($message) < 5) {
    $errors['message'] = 'Please tell us a little about what you need.';
}
$servicesChosen = [];
if ($form === 'consultation') {
    foreach ((array) ($_POST['services'] ?? []) as $s) {
        if (is_string($s) && $s !== '') {
            $servicesChosen[] = mb_substr(trim($s), 0, 80);
        }
    }
    $servicesChosen = array_slice(array_unique($servicesChosen), 0, 15);
    if (!$servicesChosen) {
        $errors['services[]'] = 'Please choose at least one area.';
    }
}
if ($form === 'career' && field('role', 120) === '') {
    $errors['role'] = 'Please choose a role or area.';
}
// Link-stuffing spam.
if (preg_match_all('#https?://|www\.#i', $message) > 3 || stripos($message, '[url=') !== false) {
    respond(false, 'Your message contains too many links. Please remove them and try again.');
}

// --- CV upload ------------------------------------------------------------------------
$cv = null;
if ($form === 'career') {
    $f = $_FILES['cv'] ?? null;
    if (!$f || ($f['error'] ?? UPLOAD_ERR_NO_FILE) === UPLOAD_ERR_NO_FILE) {
        $errors['cv'] = 'Please attach your CV.';
    } elseif ($f['error'] !== UPLOAD_ERR_OK) {
        $errors['cv'] = 'The upload failed. Please try a smaller file.';
    } else {
        $ext = strtolower(pathinfo((string) $f['name'], PATHINFO_EXTENSION));
        $mimes = [
            'pdf' => ['application/pdf'],
            'doc' => ['application/msword', 'application/x-ole-storage', 'application/CDFV2', 'application/vnd.ms-office'],
            'docx' => ['application/vnd.openxmlformats-officedocument.wordprocessingml.document', 'application/zip', 'application/octet-stream'],
        ];
        $mime = function_exists('finfo_open') ? (string) finfo_file(finfo_open(FILEINFO_MIME_TYPE), $f['tmp_name']) : '';
        if (!isset($mimes[$ext])) {
            $errors['cv'] = 'Please upload a PDF or Word document.';
        } elseif ($f['size'] > 5 * 1024 * 1024) {
            $errors['cv'] = 'The file is larger than 5 MB.';
        } elseif ($mime !== '' && !in_array($mime, $mimes[$ext], true)) {
            $errors['cv'] = 'That file does not look like a PDF or Word document.';
        } else {
            $stored = bin2hex(random_bytes(16)) . '.' . $ext;
            $dir = storage_dir('cv');
            if (@move_uploaded_file($f['tmp_name'], $dir . '/' . $stored)) {
                @chmod($dir . '/' . $stored, 0640);
                $cv = ['file' => $stored, 'name' => preg_replace('/[^A-Za-z0-9._ -]/', '_', basename((string) $f['name'])), 'mime' => $mime ?: 'application/octet-stream', 'size' => (int) $f['size']];
            } else {
                $errors['cv'] = 'We could not save your file. Please try again or e-mail your CV.';
            }
        }
    }
}

if ($errors) {
    respond(false, 'Please check the highlighted fields.', ['errors' => $errors]);
}

// --- Save -----------------------------------------------------------------------------
$now = date('Y-m-d H:i:s');
$notifyTo = array_map('trim', explode(',', (string) cfg('forms.notify_to', '')));
$brand = company_name();

if ($form === 'newsletter') {
    $exists = false;
    store_update('subscribers', function (array $rows) use ($email, $now, &$exists) {
        foreach ($rows as &$r) {
            if (strcasecmp($r['email'], $email) === 0) {
                $exists = true;
                $r['status'] = 'subscribed';
                $r['updated'] = $now;
            }
        }
        unset($r);
        if (!$exists) {
            array_unshift($rows, ['id' => new_record_id(), 'created' => $now, 'email' => $email, 'status' => 'subscribed', 'source' => field('source', 200) ?: 'website']);
        }
        return $rows;
    });
    if (!$exists) {
        send_mail($notifyTo, 'New newsletter subscriber: ' . $email, email_layout('New newsletter subscriber', '<p>' . e($email) . ' subscribed to The Veridian Brief on ' . e($now) . '.</p>'));
        if (cfg('forms.auto_reply', true)) {
            $unsub = abs_url('unsubscribe') . '?e=' . rawurlencode($email) . '&t=' . unsubscribe_token($email);
            send_mail([$email], 'Welcome to The Veridian Brief', email_layout('Welcome to The Veridian Brief', '<p>Thank you for subscribing. Once a month we will send you the tax changes, FBR notifications and compliance deadlines that matter — explained in plain language.</p><p>Meanwhile, our <a href="' . e(abs_url('knowledge-bank')) . '" style="color:#156b57">Knowledge Bank</a> and <a href="' . e(abs_url('tax-calendar')) . '" style="color:#156b57">tax calendar</a> are always available.</p><p style="font-size:12px;color:#667c76">Not for you? <a href="' . e($unsub) . '" style="color:#667c76">Unsubscribe</a>.</p>'));
        }
    }
    respond(true, $exists ? 'You are already subscribed — thank you!' : 'Thank you for subscribing. Please check your inbox.', ['title' => 'Subscribed']);
}

$record = [
    'id' => new_record_id(),
    'type' => $form,
    'created' => $now,
    'status' => 'new',
    'name' => $name,
    'email' => $email,
    'phone' => $phone,
    'company' => field('company', 160),
    'city' => field('city', 80),
    'message' => $message,
    'source' => field('source', 200),
    'ip' => client_ip(),
];
if ($form === 'contact') {
    $record['service'] = field('service', 120);
}
if ($form === 'consultation') {
    $record['service'] = implode(', ', $servicesChosen);
    foreach (['urgency' => 80, 'entity' => 80, 'industry' => 80, 'turnover' => 80, 'preferred_date' => 20, 'mode' => 40, 'time' => 40] as $k => $max) {
        $record[$k] = field($k, $max);
    }
}
if ($form === 'career') {
    $record['role'] = field('role', 120);
    $record['qualification'] = field('qualification', 160);
    $record['cv'] = $cv;
}

$storeName = $form === 'career' ? 'applications' : 'enquiries';
$saved = store_add($storeName, $record);

// --- E-mails -------------------------------------------------------------------------
$labels = [
    'Name' => $record['name'], 'E-mail' => $record['email'], 'Phone' => $record['phone'], 'Company' => $record['company'],
    'City' => $record['city'], 'Service' => $record['service'] ?? '', 'Urgency' => $record['urgency'] ?? '',
    'Business type' => $record['entity'] ?? '', 'Industry' => $record['industry'] ?? '', 'Turnover' => $record['turnover'] ?? '',
    'Preferred date' => $record['preferred_date'] ?? '', 'Meeting' => trim(($record['mode'] ?? '') . ' ' . ($record['time'] ?? '')),
    'Role' => $record['role'] ?? '', 'Qualification' => $record['qualification'] ?? '',
    'Message' => $record['message'], 'Sent from page' => $record['source'], 'Received' => $now,
];
$titles = ['contact' => 'New enquiry', 'consultation' => 'New consultation request', 'career' => 'New job application'];
$subject = $titles[$form] . ': ' . $name . (!empty($record['service']) ? ' — ' . mb_strimwidth($record['service'], 0, 60, '…') : '') . (!empty($record['role']) ? ' — ' . $record['role'] : '');
$adminLink = abs_url('admin/') . '?tab=' . $storeName;
$body = email_fields_table($labels)
    . '<p><a href="mailto:' . e($email) . '" style="display:inline-block;background:#0f5143;color:#fff;padding:10px 18px;border-radius:999px;text-decoration:none;font-weight:bold">Reply to ' . e($name) . '</a></p>'
    . '<p style="font-size:13px;color:#667c76">All enquiries are also saved in your website admin area: <a href="' . e($adminLink) . '" style="color:#156b57">' . e($adminLink) . '</a></p>';
$attachments = $cv ? [[storage_dir('cv') . '/' . $cv['file'], $cv['name'], $cv['mime']]] : [];
$mailed = send_mail($notifyTo, $subject, email_layout($titles[$form], $body), ['reply_to' => $email, 'attachments' => $attachments]);

if (!$saved && !$mailed) {
    log_line('mail-log', 'Enquiry could not be saved or e-mailed: ' . $email);
    respond(false, 'Sorry — we could not send your message. Please e-mail us at ' . contact_email() . '.');
}

if (cfg('forms.auto_reply', true)) {
    $first = e(strtok($name, ' ') ?: $name);
    $replies = [
        'contact' => ['We have received your message', '<p>Dear ' . $first . ',</p><p>Thank you for contacting ' . e($brand) . '. A senior adviser will review your message and reply within one business day.</p><p>If your matter is urgent — for example an FBR notice with a close deadline — simply reply to this e-mail and let us know.</p>'],
        'consultation' => ['Your consultation request', '<p>Dear ' . $first . ',</p><p>Thank you for booking a consultation with ' . e($brand) . '. We will confirm the date and time within one business day.</p><p>To make the most of the call, you may want to have to hand any recent notices, returns or documents relating to your question.</p>'],
        'career' => ['Thank you for your application', '<p>Dear ' . $first . ',</p><p>Thank you for your interest in joining ' . e($brand) . '. We review every application carefully and will contact you if there is a suitable opportunity.</p>'],
    ];
    [$t, $c] = $replies[$form];
    send_mail([$email], $t . ' — ' . $brand, email_layout($t, $c . '<p>Kind regards,<br>' . e($brand) . '</p><p style="font-size:12px;color:#667c76">This is an automatic confirmation. Please do not send confidential documents by e-mail until we have agreed a secure method.</p>'), ['reply_to' => contact_email()]);
}

$done = [
    'contact' => ['Thank you — message received.', 'A senior adviser will reply within one business day. A confirmation is on its way to your inbox.'],
    'consultation' => ['Your consultation request is in.', 'We will confirm the date and time within one business day. A confirmation is on its way to your inbox.'],
    'career' => ['Application received — thank you.', 'We review every application carefully and will be in touch if there is a suitable role.'],
];
respond(true, $done[$form][1], ['title' => $done[$form][0]]);

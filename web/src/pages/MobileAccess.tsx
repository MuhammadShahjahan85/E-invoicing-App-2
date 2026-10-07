// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useEffect, useState } from 'react'
import { api } from '../api'
import { Spinner, useLoad } from '../components/ui'
import { canInstall, isAndroid, isIOS, isStandalone, onInstallChange, promptInstall } from '../pwa'
import { useSession, useToast } from '../state'

interface Addresses {
  urls: string[]
  https: boolean
  localCA: boolean
}

export default function MobileAccess() {
  const s = useSession()
  const toast = useToast()
  const { data } = useLoad(() => api.get<Addresses>('/system/addresses'), [])
  const [installable, setInstallable] = useState(canInstall())
  const [selected, setSelected] = useState('')
  useEffect(() => onInstallChange(() => setInstallable(canInstall())), [])

  const here = window.location.origin + '/'
  const onLocalhost = /^(localhost|127\.0\.0\.1|\[::1\])$/.test(window.location.hostname)
  const urls = data ? Array.from(new Set([...(onLocalhost ? [] : [here]), ...data.urls])) : []
  const shown = selected || urls[0] || here
  const standalone = isStandalone()

  const install = async () => {
    if (await promptInstall()) toast('ok', `${s.meta.product} installed on this device`)
  }

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Mobile app & access</h1>
          <p>Use {s.meta.product} on phones and tablets as an installed app, or in any browser, from computers on your office network.</p>
        </div>
      </div>

      {standalone && <div className="alert alert-ok">You are using the installed app on this device.</div>}

      <div className="grid g2">
        <div className="card card-pad">
          <h3>1. Open on your phone</h3>
          <p className="small muted">Connect the phone to the office Wi-Fi, open the camera and scan the code. It opens {s.meta.product} in the phone's browser.</p>
          {!data ? (
            <Spinner />
          ) : (
            <div className="row" style={{ alignItems: 'flex-start' }}>
              <div className="connect-qr">
                <img src={`/api/v1/system/link-qr.svg?url=${encodeURIComponent(shown)}`} alt={`QR code for ${shown}`} />
              </div>
              <div className="stack" style={{ minWidth: 0, flex: 1 }}>
                <div className="mono small" style={{ wordBreak: 'break-all' }}>
                  {shown}
                </div>
                {urls.length > 1 && (
                  <select value={shown} onChange={(e) => setSelected(e.target.value)} aria-label="Server address">
                    {urls.map((u) => (
                      <option key={u}>{u}</option>
                    ))}
                  </select>
                )}
                {onLocalhost && urls.length === 0 && (
                  <div className="alert alert-warn small">
                    This computer has no network address. Connect it to the office network so phones can reach it.
                  </div>
                )}
              </div>
            </div>
          )}
        </div>

        <div className="card card-pad">
          <h3>2. Install the app</h3>
          <p className="small muted">
            {s.meta.product} installs from the browser like an app: its own icon, full screen, no app store needed. Updates arrive automatically when the server is updated.
          </p>
          {installable ? (
            <button className="btn btn-primary" onClick={install}>
              Install on this device
            </button>
          ) : standalone ? null : (
            <ol className="small">
              {isIOS() ? (
                <>
                  <li>Open this page in <b>Safari</b>.</li>
                  <li>
                    Tap the <b>Share</b> button, then <b>Add to Home Screen</b>, then <b>Add</b>.
                  </li>
                </>
              ) : isAndroid() ? (
                <>
                  <li>Open this page in <b>Chrome</b>.</li>
                  <li>
                    Tap the <b>⋮</b> menu, then <b>Install app</b> (or <b>Add to Home screen</b>).
                  </li>
                </>
              ) : (
                <>
                  <li>
                    In <b>Chrome</b> or <b>Edge</b>, click the install icon at the right of the address bar (or menu → <b>Install {s.meta.product}</b>).
                  </li>
                  <li>On phones: Android Chrome → ⋮ → Install app; iPhone Safari → Share → Add to Home Screen.</li>
                </>
              )}
            </ol>
          )}
          {!window.isSecureContext && (
            <div className="alert alert-warn small">
              Installing needs a secure (HTTPS) connection that the device trusts. Complete step 3 on the phone first, then reopen the page.
            </div>
          )}
        </div>

        <div className="card card-pad">
          <h3>3. Trust this server on each phone (once)</h3>
          <p className="small muted">
            The server protects all traffic with HTTPS using its own certificate authority. Installing that certificate once on a phone removes security warnings and allows the app
            to be installed.
          </p>
          {window.location.protocol !== 'https:' && (
            <div className="alert alert-warn small">
              This page is open over plain HTTP. Phones can use the system in the browser, but installing it as an app needs HTTPS: run the server with HTTPS
              (the default; remove <span className="mono">--no-tls</span>) or behind an HTTPS reverse proxy.
            </div>
          )}
          {data?.localCA ? (
            <a className="btn" href="/api/v1/system/ca.crt">
              Download certificate
            </a>
          ) : window.location.protocol === 'https:' ? (
            <p className="small">This server uses a certificate issued by a public or company certificate authority; no download is needed.</p>
          ) : null}
          <div className="grid g2 mt">
            <div className="small">
              <b>Android</b>
              <ol>
                <li>Download the certificate on the phone.</li>
                <li>Settings → Security → More security settings → Encryption & credentials → Install a certificate → CA certificate.</li>
                <li>Select the downloaded file and confirm.</li>
              </ol>
            </div>
            <div className="small">
              <b>iPhone / iPad</b>
              <ol>
                <li>Download the certificate in Safari and allow the profile download.</li>
                <li>Settings → Profile Downloaded → Install.</li>
                <li>Settings → General → About → Certificate Trust Settings → turn on full trust for it.</li>
              </ol>
            </div>
          </div>
        </div>

        <div className="card card-pad">
          <h3>Using it outside the office</h3>
          <p className="small">
            The system runs on your own server, so phones reach it over the office network. For staff working elsewhere, connect them securely instead of opening the server to the
            internet:
          </p>
          <ul className="small">
            <li>
              <b>VPN</b> to the office network (router or firewall VPN, or a mesh VPN such as Tailscale or ZeroTier).
            </li>
            <li>
              <b>Secure tunnel or reverse proxy</b> with a public domain name and a trusted certificate (for example Cloudflare Tunnel, or Caddy/Nginx with Let's Encrypt). Add
              the domain name to <span className="mono">tls.hosts</span> in config.json if the server's own certificate is used behind it.
            </li>
          </ul>
          <p className="small muted">Never forward port 8443 straight from the internet to the server. The installation guide describes each option.</p>
        </div>
      </div>
    </>
  )
}

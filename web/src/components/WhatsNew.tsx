// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useEffect, useState } from 'react'
import { Sparkles } from 'lucide-react'
import { api } from '../api'
import { Modal } from './ui'
import { ReleaseNotes } from '../pages/settings/SoftwareUpdates'
import type { UpdateResult } from '../types'

const seenKey = 'whats-new-seen'

/** WhatsNew shows each user, once per browser, what the latest update brought. */
export function WhatsNew() {
  const [u, setU] = useState<UpdateResult | null>(null)
  useEffect(() => {
    api
      .get<{ version: string; update: UpdateResult | null }>('/system/whats-new')
      .then((r) => {
        if (!r.update?.notes) return
        let seen = ''
        try {
          seen = localStorage.getItem(seenKey) ?? ''
        } catch {
          // storage unavailable: show the note this time
        }
        if (seen !== r.update.to) setU(r.update)
      })
      .catch(() => {})
  }, [])
  if (!u) return null
  const close = () => {
    try {
      localStorage.setItem(seenKey, u.to)
    } catch {
      // storage unavailable
    }
    setU(null)
  }
  return (
    <Modal
      title={`What's new in version ${u.to}`}
      onClose={close}
      footer={
        <button className="btn btn-primary" onClick={close}>
          Got it
        </button>
      }
    >
      <p className="whats-new-lead">
        <Sparkles size={18} /> The software was updated from version {u.from} to {u.to}.
      </p>
      <ReleaseNotes text={u.notes ?? ''} />
    </Modal>
  )
}

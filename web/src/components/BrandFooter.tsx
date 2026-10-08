// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useEffect, useState } from 'react'
import { api } from '../api'

/** BrandFooter shows the developer credit and copyright on pages shown before login. */
export default function BrandFooter() {
  const [info, setInfo] = useState<{ developedBy?: string; copyright?: string }>({})
  useEffect(() => {
    api
      .get<{ developedBy: string; copyright: string }>('/system/status')
      .then(setInfo)
      .catch(() => {})
  }, [])
  if (!info.copyright) return null
  return (
    <p className="brand-footer">
      {info.developedBy}
      <br />
      {info.copyright}
    </p>
  )
}

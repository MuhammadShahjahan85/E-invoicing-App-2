// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Pakistan's main cities and towns with their province (FBR's names), to
// fill in the destination of supply from a city.

const byProvince: Record<string, string[]> = {
  PUNJAB: ['Lahore', 'Faisalabad', 'Rawalpindi', 'Gujranwala', 'Multan', 'Sialkot', 'Bahawalpur', 'Sargodha', 'Sheikhupura', 'Jhang', 'Rahim Yar Khan',
    'Gujrat', 'Kasur', 'Sahiwal', 'Okara', 'Wah Cantt', 'Dera Ghazi Khan', 'Chiniot', 'Kamoke', 'Mandi Bahauddin', 'Jhelum', 'Sadiqabad', 'Khanewal',
    'Hafizabad', 'Muzaffargarh', 'Khanpur', 'Attock', 'Chakwal', 'Vehari', 'Mianwali', 'Bahawalnagar', 'Narowal', 'Pakpattan', 'Lodhran',
    'Toba Tek Singh', 'Taxila', 'Murree', 'Raiwind', 'Daska', 'Wazirabad', 'Burewala', 'Jaranwala'],
  SINDH: ['Karachi', 'Hyderabad', 'Sukkur', 'Larkana', 'Nawabshah', 'Mirpur Khas', 'Jacobabad', 'Shikarpur', 'Khairpur', 'Dadu', 'Thatta', 'Badin',
    'Kotri', 'Tando Adam', 'Tando Allahyar', 'Umerkot', 'Ghotki', 'Sanghar', 'Nooriabad'],
  'KHYBER PAKHTUNKHWA': ['Peshawar', 'Mardan', 'Mingora', 'Swat', 'Kohat', 'Abbottabad', 'Dera Ismail Khan', 'Mansehra', 'Nowshera', 'Charsadda',
    'Swabi', 'Bannu', 'Haripur', 'Chitral', 'Hangu', 'Karak', 'Timergara', 'Battagram', 'Lakki Marwat'],
  BALOCHISTAN: ['Quetta', 'Gwadar', 'Turbat', 'Khuzdar', 'Hub', 'Chaman', 'Sibi', 'Zhob', 'Loralai', 'Dera Murad Jamali', 'Uthal'],
  'CAPITAL TERRITORY': ['Islamabad'],
  'AZAD JAMMU AND KASHMIR': ['Muzaffarabad', 'Mirpur', 'Kotli', 'Bhimber', 'Bagh', 'Rawalakot'],
  'GILGIT BALTISTAN': ['Gilgit', 'Skardu', 'Hunza', 'Chilas'],
}

const index = new Map<string, string>()
for (const [p, cities] of Object.entries(byProvince)) {
  for (const c of cities) index.set(c.toLowerCase(), p)
}

/** pkCities is the list offered when typing a city. */
export const pkCities = Object.values(byProvince).flat().sort()

/** provinceForCity returns FBR's province name for a known city, or "". */
export function provinceForCity(city: string): string {
  return index.get(city.trim().toLowerCase()) ?? ''
}

/** regNoHint explains an NTN/CNIC as typed: "CNIC 35202-1234567-1". */
export function regNoHint(v: string): string {
  const d = v.replace(/\D/g, '')
  if (d.length === 13) return `CNIC ${d.slice(0, 5)}-${d.slice(5, 12)}-${d.slice(12)}`
  if (d.length === 7) return `NTN ${d}`
  if (d.length === 9) return `9-digit registration number ${d}`
  if (d.length === 8) return 'An NTN has 7 digits — leave out the check digit after the dash'
  if (d.length > 0) return `${d.length} digits — an NTN has 7 and a CNIC 13`
  return ''
}

/** provinceFromAddress finds a known city in an address ("Saddar, Karachi" → SINDH). */
export function provinceFromAddress(addr: string): string {
  const words = addr.toLowerCase().match(/[a-z]+/g) ?? []
  for (let i = words.length - 1; i >= 0; i--) {
    for (let n = 3; n >= 1; n--) {
      if (i - n + 1 < 0) continue
      const p = index.get(words.slice(i - n + 1, i + 1).join(' '))
      if (p) return p
    }
  }
  return ''
}

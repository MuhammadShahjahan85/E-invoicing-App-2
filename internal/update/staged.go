// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package update

// Staged is a downloaded and verified update, ready to install.
type Staged struct {
	Dir      string // <data>/updates/<version>, which also holds update.json
	Path     string // the installer
	Manifest *Manifest
	File     File
}

// RequestName is the file by which the service asks the Linux update
// helper to install a staged update (in <data>/updates).
const RequestName = "install-request.json"

// ResultName is the file in which the Linux update helper reports a failure.
const ResultName = "install-result.json"

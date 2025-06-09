// Copyright 2012-2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"
)

//go:embed COREBOOTCONFIG
var corebootconfig []byte
//go:embed COREBOOTCUSTOM
var corebootcustom []byte

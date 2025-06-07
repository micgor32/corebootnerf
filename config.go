// Copyright 2012-2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	//"os"
	_ "embed"
)

//go:embed CONFIG
var linuxconfig []byte
//go:embed COREBOOTCONFIG 
var corebootconfig []byte

// func genConfig() error {
// 	var err error
// 	linuxconfig, err = os.ReadFile("CONFIG")
// 	if err != nil {
// 		return err
// 	}
	
// 	corebootconfig, err = os.ReadFile("COREBOOTCONFIG")
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

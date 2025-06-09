// Copyright 2012-2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"github.com/go-ini/ini"
	"flag"
	"errors"
)

var (
	configTxt = `loglevel=1
	init=/init
rootwait
`
	deps          = flag.Bool("deps", false, "apt/pacman/dnf all the things we need")
	fetch         = flag.Bool("fetch", false, "Fetch all the things we need")
	customkern    = flag.String("customkern", "", "Path to the custom kernel")
	initramfs     = flag.String("initramfs", "", "Path to the custom initramfs")
	extra         = flag.String("extra", "", "Comma-separated list of extra packages to include")
	corebootVer   = "25.03"
	workingDir    = ""
	homeDir       = ""
	threads       = runtime.NumCPU() + 2 // Number of threads to use when calling make.
	packageListDebian = []string{ 
		"bison",
		"git",
		"golang",
		"build-essential",
		"curl",
		"gnat",
		"flex",
		"gnat",
		"libncurses-dev",
		"libssl-dev",
		"zlib1g-dev",
		"pkgconf",
	}
	packageListArch = []string{
		"base-devel",
		"curl",
		"git",
		"gcc-ada",
		"ncurses",
		"wget",
		"zlib",
	}
	packageListRedhat = []string{
		"git",
		"make",
		"gcc-gnat",
		"flex",
		"bison",
		"xz",
		"bzip2",
		"gcc",
		"g++",
		"ncurses-devel",
		"wget",
		"zlib-devel",
		"patch",
	}
)

func cp(inputLoc string, outputLoc string) error {
	// Don't check for an error, there are all kinds of
	// reasons a remove can fail even if the file is
	// writeable
	os.Remove(outputLoc)

	if _, err := os.Stat(inputLoc); err != nil {
		return err
	}
	fileContent, err := os.ReadFile(inputLoc)
	if err != nil {
		return err
	}
	return os.WriteFile(outputLoc, fileContent, 0777)
}

func corebootGet() error {
	baseUrl := "https://coreboot.org/releases/coreboot-"
	fullUrl := baseUrl + corebootVer + ".tar.xz"
	var args = []string{fullUrl}
	fmt.Printf("-------- Getting coreboot via wget %v\n", fullUrl)
	cmd := exec.Command("wget", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("didn't wget coreboot %v", err)
		return err
	}
	cmd = exec.Command("tar", "xvf", "coreboot-" + corebootVer + ".tar.xz")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("untar failed %v", err)
		return err
	}
	cmd = exec.Command("make", "-j"+strconv.Itoa(threads), "crossgcc-i386", "iasl")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Dir = "coreboot-" + corebootVer
	if err := cmd.Run(); err != nil {
		fmt.Printf("untar failed %v", err)
		return err
	}
	return nil
}

func customKern() error {
	if *initramfs == "" {
		return errors.New("initramfs not provided")
	}

	custom := fmt.Appendf(
		corebootcustom,
		"\nCONFIG_PAYLOAD_FILE=\"%s\"\nCONFIG_LINUX_INITRD=\"%s\"",
		*customkern,
		*initramfs,
	)

	if err := os.WriteFile("coreboot-" + corebootVer +"/.config", custom, 0666); err != nil {
		fmt.Printf("writing corebootconfig: %v", err)
		return err
	}

	return nil
}

func buildCoreboot() error {
	if *customkern != "" {
		if err := customKern(); err != nil {
			return err
		}
	} else {
		if err := os.WriteFile("coreboot-" + corebootVer +"/.config", []byte(corebootconfig), 0666); err != nil {
			fmt.Printf("writing corebootconfig: %v", err)
			return err
		}
	}

	cmd := exec.Command("make", "-j"+strconv.Itoa(threads))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append(os.Environ(), "ARCH=x86_64")
	cmd.Dir = "coreboot-" + corebootVer
	err := cmd.Run()
	if err != nil {
		return err
	}
	if _, err := os.Stat("coreboot-" + corebootVer + "/build/coreboot.rom"); err != nil {
		return err
	}
	fmt.Printf("coreboot.rom created")
	return nil
}

func check() error {
	if os.Getenv("GOPATH") == "" {
		return fmt.Errorf("You have to set GOPATH.")
	}
	return nil
}

func pacmaninstall() error {
	missing := []string{}
	for _, packageName := range packageListArch {
		cmd := exec.Command("pacman", "-Ql", packageName)
		if err := cmd.Run(); err != nil {
			missing = append(missing, packageName)
		}
	}

	if len(missing) == 0 {
		fmt.Println("No missing dependencies to install")
		return nil
	}

	fmt.Printf("Using pacman to get %v\n", missing)
	get := []string{"pacman", "-S", "--noconfirm"}
	get = append(get, missing...)
	cmd := exec.Command("sudo", get...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}


func dnfinstall() error {
	missing := []string{}
	for _, packageName := range packageListRedhat {
		cmd := exec.Command("dnf", "info", packageName)
		if err := cmd.Run(); err != nil {
			missing = append(missing, packageName)
		}
	}

	if len(missing) == 0 {
		fmt.Println("No missing dependencies to install")
		return nil
	}

	fmt.Printf("Using dnf to get %v\n", missing)
	get := []string{"dnf", "-y", "install"}
	get = append(get, missing...)
	cmd := exec.Command("sudo", get...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func aptget() error {
	missing := []string{}
	for _, packageName := range packageListDebian {
		cmd := exec.Command("dpkg", "-s", packageName)
		if err := cmd.Run(); err != nil {
			missing = append(missing, packageName)
		}
	}

	if len(missing) == 0 {
		fmt.Println("No missing dependencies to install")
		return nil
	}

	fmt.Printf("Using apt-get to get %v\n", missing)
	get := []string{"apt-get", "-y", "install"}
	get = append(get, missing...)
	cmd := exec.Command("sudo", get...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()

}

func depinstall() error {
	cfg, err := ini.Load("/etc/os-release")
    if err != nil {
        log.Fatal("Fail to read file: %v\n", err)
    }

    ConfigParams := make(map[string]string)
    ConfigParams["ID"] = cfg.Section("").Key("ID").String()
	osID := ConfigParams["ID"]

	switch osID {
		case "fedora":
			dnfinstall()
		case "rhel":
			dnfinstall()
		case "debian":
			aptget()
		case "ubuntu":
			aptget()
		case "arch":
			pacmaninstall()
		default:
			log.Fatal("No matching OS found\n")
	}

	return nil
}

func allFunc() error {
	var cmds = []struct {
		f      func() error
		skip   bool
		ignore bool
		n      string
	}{
		{f: check, skip: false, ignore: false, n: "check environment"},
		{f: depinstall, skip: !*deps, ignore: false, n: "install depenedencies"},
		{f: corebootGet, skip: !*fetch, ignore: false, n: "Git clone coreboot"},
		{f: buildCoreboot, skip: *deps, ignore: false, n: "build coreboot"},
	}

	for _, c := range cmds {
		log.Printf("-----> Step %v: ", c.n)
		if c.skip {
			log.Printf("-------> Skip")
			continue
		}
		log.Printf("----------> Start")
		err := c.f()
		if c.ignore {
			log.Printf("----------> Ignore result")
			continue
		}
		if err != nil {
			return fmt.Errorf("%v: %v", c.n, err)
		}
		log.Printf("----------> Finished %v\n", c.n)
	}
	return nil
}

func main() {
	flag.Parse()
	log.Printf("Building coreboot verstion %v\n", corebootVer)
	if err := allFunc(); err != nil {
		log.Fatalf("fail error is : %v", err)
	}
	log.Printf("execution completed successfully\n")
}

//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/magefile/mage/sh"
)

var (
	appName  = "filestore"
	mainPath = "./cmd/server"
	buildDir = "build"
	modulePath = "filestore"
)

// 目标平台列表
var platforms = []struct {
	os   string
	arch string
}{
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"windows", "amd64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
}

// ldflags 返回注入版本信息的 ldflags 参数
func ldflags() string {
	return fmt.Sprintf("-s -w -X %s/internal/config.Version=dev", modulePath)
}

// Build 编译当前平台的二进制文件
func Build() error {
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return err
	}
	out := filepath.Join(buildDir, binaryName(runtime.GOOS, runtime.GOARCH))
	fmt.Printf("Building %s ...\n", out)
	return sh.Run("go", "build", "-ldflags", ldflags(), "-o", out, mainPath)
}

// BuildAll 编译所有目标平台的二进制文件
func BuildAll() error {
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return err
	}
	for _, p := range platforms {
		out := filepath.Join(buildDir, binaryName(p.os, p.arch))
		fmt.Printf("Building %s ...\n", out)
		env := map[string]string{
			"GOOS":   p.os,
			"GOARCH": p.arch,
		}
		if err := sh.RunWith(env, "go", "build", "-ldflags", ldflags(), "-o", out, mainPath); err != nil {
			return fmt.Errorf("build %s/%s failed: %w", p.os, p.arch, err)
		}
	}
	fmt.Println("All binaries built successfully.")
	return nil
}

// Clean 清理构建产物
func Clean() error {
	fmt.Println("Cleaning build directory ...")
	return sh.Rm(buildDir)
}

// Run 编译并运行当前平台的二进制文件
func Run() error {
	if err := Build(); err != nil {
		return err
	}
	out := filepath.Join(buildDir, binaryName(runtime.GOOS, runtime.GOARCH))
	return sh.RunV(out)
}

// Tidy 整理 Go 模块依赖
func Tidy() error {
	return sh.Run("go", "mod", "tidy")
}

// Vet 对代码进行静态检查
func Vet() error {
	return sh.Run("go", "vet", "./...")
}

// Test 运行单元测试
func Test() error {
	return sh.Run("go", "test", "-v", "./...")
}

// binaryName 根据平台生成二进制文件名
func binaryName(goos, arch string) string {
	name := fmt.Sprintf("%s-%s-%s", appName, goos, arch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

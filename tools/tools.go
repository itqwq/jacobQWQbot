//go:build tools
// +build tools

package tools

/*
具体来说，这个文件的作用是引入了一个名为 github.com/vektra/mockery/v2 的工具。通过导入这个工具，它可以确保在构建项目时，这个工具会被下载和安装，以便在项目中使用。通常，这样的文件用于管理项目的开发和构建过程中使用的一些工具或辅助程序。
*/
import (
	_ "github.com/vektra/mockery/v2"
)

package main

import (
	"flag"

	"github.com/sirupsen/logrus"
)

func main() {
	port := flag.String("port", ":18688", "服务端口")
	debug := flag.Bool("debug", false, "是否开启调试日志")
	flag.Parse()

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	logrus.Infof("1688-MCP 启动中... port=%s", *port)

	service, err := NewAlibabaService()
	if err != nil {
		logrus.Fatalf("服务初始化失败: %v", err)
	}

	appServer := NewAppServer(service)
	if err := appServer.Start(*port); err != nil {
		logrus.Fatalf("服务启动失败: %v", err)
	}
}

// Copyright 2023 LiveKit, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/livekit/livekit-server/pkg/config"
	"github.com/livekit/livekit-server/pkg/logger"
	"github.com/livekit/livekit-server/pkg/server"
	"github.com/livekit/livekit-server/version"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	app := &cli.App{
		Name:        "livekit-server",
		Usage:       "High performance WebRTC server",
		Version:     version.Version,
		Description: "LiveKit is an open source WebRTC infrastructure for building real-time audio/video applications.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "path to LiveKit config file",
				EnvVars: []string{"LIVEKIT_CONFIG_FILE"},
			},
			&cli.StringFlag{
				Name:    "config-body",
				Usage:   "LiveKit config in YAML, read from stdin",
				EnvVars: []string{"LIVEKIT_CONFIG_BODY"},
			},
			&cli.StringFlag{
				Name:    "key-file",
				Usage:   "path to file that contains API keys/secrets",
				EnvVars: []string{"LIVEKIT_KEY_FILE"},
			},
			&cli.StringFlag{
				Name:    "keys",
				Usage:   "api keys (key: secret\nkey2: secret2)",
				EnvVars: []string{"LIVEKIT_KEYS"},
			},
			&cli.StringFlag{
				Name:    "node-ip",
				Usage:   "IP address of the node, used to advertise to clients",
				EnvVars: []string{"NODE_IP"},
			},
			&cli.StringFlag{
				Name:    "redis",
				Usage:   "Redis URL (redis://[user:password@]host:port/db)",
				EnvVars: []string{"REDIS_URL"},
			},
			&cli.BoolFlag{
				Name:  "dev",
				Usage: "run in development mode (insecure, uses default keys)",
			},
		},
		Action: startServer,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func startServer(c *cli.Context) error {
	conf, err := config.NewConfig(c.String("config"), c.String("config-body"), c)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger.InitFromConfig(&conf.Logging, "livekit")

	s, err := server.InitializeServer(conf, c.String("node-ip"))
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	sigChan := make(chan os.Signal, 1)
	// Also handle SIGHUP so the process can be gracefully restarted by process
	// managers (e.g. systemd) without dropping active participant sessions.
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)

	go func() {
		sig := <-sigChan
		logger.Infow("exit requested, shutting down", "signal", sig)
		s.Stop(false)
	}()

	return s.Start()
}

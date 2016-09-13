package main

import (
	"log"
	"net"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var options struct {
	websocketOrigin    string
	serverType         string
	concurrent         int
	sampleSize         int
	stepSize           int
	limitPercentile    int
	limitRTT           time.Duration
	payloadPaddingSize int
	localAddrs         []string
}

func main() {

	rootCmd := &cobra.Command{Use: "websocket-bench", Short: "websocket benchmark tool"}
	rootCmd.PersistentFlags().StringVarP(&options.websocketOrigin, "origin", "o", "", "websocket origin")
	rootCmd.PersistentFlags().StringSliceVarP(&options.localAddrs, "local-addr", "l", []string{}, "local IP address to connect from")
	rootCmd.PersistentFlags().StringVarP(&options.serverType, "server-type", "", "standard", "server type to connect to (standard, actioncable, phoenix)")

	cmdEcho := &cobra.Command{
		Use:   "echo URL",
		Short: "Echo stress test",
		Long:  "Stress test 1 to 1 performance with an echo test",
		Run:   Stress,
	}
	cmdEcho.Flags().IntVarP(&options.concurrent, "concurrent", "c", 50, "concurrent echo requests")
	cmdEcho.Flags().IntVarP(&options.sampleSize, "sample-size", "s", 10000, "number of echoes in a sample")
	cmdEcho.Flags().IntVarP(&options.stepSize, "step-size", "", 5000, "number of clients to increase each step")
	cmdEcho.Flags().IntVarP(&options.limitPercentile, "limit-percentile", "", 95, "round-trip time percentile to for limit")
	cmdEcho.Flags().IntVarP(&options.payloadPaddingSize, "payload-padding", "", 0, "payload padding size")
	cmdEcho.Flags().DurationVarP(&options.limitRTT, "limit-rtt", "", time.Millisecond*500, "Max RTT at limit percentile")
	rootCmd.AddCommand(cmdEcho)

	cmdBroadcast := &cobra.Command{
		Use:   "broadcast URL",
		Short: "Broadcast stress test",
		Long:  "Stress test 1 to many performance with an broadcast test",
		Run:   Stress,
	}
	cmdBroadcast.Flags().IntVarP(&options.concurrent, "concurrent", "c", 4, "concurrent broadcast requests")
	cmdBroadcast.Flags().IntVarP(&options.sampleSize, "sample-size", "s", 20, "number of broadcasts in a sample")
	cmdBroadcast.Flags().IntVarP(&options.stepSize, "step-size", "", 5000, "number of clients to increase each step")
	cmdBroadcast.Flags().IntVarP(&options.limitPercentile, "limit-percentile", "", 95, "round-trip time percentile to for limit")
	cmdBroadcast.Flags().IntVarP(&options.payloadPaddingSize, "payload-padding", "", 0, "payload padding size")
	cmdBroadcast.Flags().DurationVarP(&options.limitRTT, "limit-rtt", "", time.Millisecond*500, "Max RTT at limit percentile")
	rootCmd.AddCommand(cmdBroadcast)

	rootCmd.Execute()
}

func Stress(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		cmd.Help()
		os.Exit(1)
	}

	config := &BenchmarkConfig{}
	config.WebsocketURL = args[0]
	config.WebsocketOrigin = options.websocketOrigin
	switch cmd.Name() {
	case "echo":
		config.ClientCmd = clientEchoCmd
	case "broadcast":
		config.ClientCmd = clientBroadcastCmd
	default:
		panic("invalid command name")
	}
	config.PayloadPaddingSize = options.payloadPaddingSize
	config.StepSize = options.stepSize
	config.Concurrent = options.concurrent
	config.SampleSize = options.sampleSize
	config.LimitPercentile = options.limitPercentile
	config.LimitRTT = options.limitRTT
	config.ResultRecorder = &TextResultRecorder{w: os.Stdout}

	localAddrs := parseTCPAddrs(options.localAddrs)
	for _, a := range localAddrs {
		config.ClientFactories = append(config.ClientFactories, &clientFactory{laddr: a})
	}
	b := NewBenchmark(config)
	err := b.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func parseTCPAddrs(stringAddrs []string) []*net.TCPAddr {
	var tcpAddrs []*net.TCPAddr
	for _, s := range stringAddrs {
		tcpAddrs = append(tcpAddrs, &net.TCPAddr{IP: net.ParseIP(s)})
	}

	if len(tcpAddrs) == 0 {
		tcpAddrs = []*net.TCPAddr{nil}
	}

	return tcpAddrs
}

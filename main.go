package main

import (
	"flag"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/go-ini/ini"
	"log"
	"os"
	"os/exec"
	"os/signal"
)

var (
	conf  string
	shell string
)

func main() {
	// Parse flag
	flag.StringVar(&conf, "c", "config.ini", "config.ini path")
	flag.StringVar(&shell, "s", "bash", "shell name")
	flag.Parse()

	cfg, err := ini.Load(conf)
	if err != nil {
		log.Fatal(err)
	}

	// Create Client
	mqttConnect(cfg, shell)

	// Wait exit signal
	console := make(chan os.Signal, 1)
	signal.Notify(console, os.Interrupt)
	<-console
}

func mqttConnect(cfg *ini.File, shell string) {
	broker := cfg.Section("").Key("broker").String()
	// Client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(cfg.Section("").Key("clientId").String())
	if cfg.Section("").HasValue("username") {
		opts.SetUsername(cfg.Section("").Key("username").String())
		opts.SetUsername(cfg.Section("").Key("password").String())
	}

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		fmt.Printf("[-] Connect lost: %v", err)
		mqttConnect(cfg, shell)
	})

	// Main Handler
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		section := msg.Topic()
		key := string(msg.Payload())
		fmt.Printf("[+] Received message: %s from topic: %s\n", key, section)

		if cfg.HasSection(section); cfg.Section(section).HasKey(key) {
			command := cfg.Section(section).Key(key).String()
			cmd := exec.Command(shell, argCombine(shell), command)
			out, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Println(out)
				log.Fatal(err)
			}
			fmt.Printf("[+] %s\n%s\n", command, out)
		}
	})

	opts.SetOnConnectHandler(func(client mqtt.Client) {
		fmt.Println("[+] Connect", broker)
	})
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	// Subscribe topics
	for index, topic := range cfg.SectionStrings() {
		if index > 0 {
			Subscribe(client, topic)
		}
	}

	//client.Disconnect(250)
}

func Subscribe(client mqtt.Client, topic string) {
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	fmt.Println("[+] Subscribe topic:", topic)
}

func argCombine(shell string) string {
	if shell == "cmd" {
		return "/C"
	} else {
		return "-c"
	}

}

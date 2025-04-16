package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"time"
)

type discordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type discordEmbed struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Color       int                 `json:"color"`
	Fields      []discordEmbedField `json:"fields"`
	Timestamp   string              `json:"timestamp"`
}

type discordWebhookPayload struct {
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Content   string         `json:"content,omitempty"`
	Embeds    []discordEmbed `json:"embeds"`
}

func SendDiscordNotification(webhookURL string, topTalkers map[string]float64, thresholdMbps float64, intervalSeconds int) error {
	if webhookURL == "" {
		return fmt.Errorf("webhook URL is empty, skipping notification")
	}

	type ipSpeed struct {
		IP    string
		Speed float64
	}
	var sortedTalkers []ipSpeed
	for ip, speed := range topTalkers {
		sortedTalkers = append(sortedTalkers, ipSpeed{IP: ip, Speed: speed})
	}
	sort.Slice(sortedTalkers, func(i, j int) bool {
		return sortedTalkers[i].Speed > sortedTalkers[j].Speed
	})

	var fields []discordEmbedField
	totalSpeed := 0.0
	for _, talker := range sortedTalkers {
		fields = append(fields, discordEmbedField{
			Name:   talker.IP,
			Value:  fmt.Sprintf("%.2f Mbps", talker.Speed),
			Inline: true,
		})
		totalSpeed += talker.Speed
	}

	embed := discordEmbed{
		Title: "🚨 Network Threshold Exceeded!",
		Description: fmt.Sprintf("Overall speed exceeded %.2f Mbps threshold (Total: %.2f Mbps) in the last %d seconds.\nTop %d talkers:",
			thresholdMbps, totalSpeed, intervalSeconds, len(sortedTalkers)),
		Color:     15158332,
		Fields:    fields,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	payload := discordWebhookPayload{
		Username: "Network Monitor",
		Embeds:   []discordEmbed{embed},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal discord payload: %w", err)
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send discord notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {

		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received non-2xx status code from discord: %d %s - %s", resp.StatusCode, resp.Status, string(bodyBytes))

	}

	fmt.Println("Successfully sent notification to Discord.")
	return nil
}

func SendInitNotification(webhookURL, interfaceName string, thresholdMbps float64, intervalSeconds int) error {
	if webhookURL == "" {
		log.Println("Webhook URL is empty, skipping initialization notification.")
		return nil
	}

	description := fmt.Sprintf(
		"Network Monitor started.\nMonitoring Interface: **%s**\nThreshold: **%.2f Mbps**\nCheck Interval: **%ds**",
		interfaceName, thresholdMbps, intervalSeconds,
	)
	if interfaceName == "" {
		description = fmt.Sprintf(
			"Network Monitor started.\nMonitoring Interface: **Auto-Selected**\nThreshold: **%.2f Mbps**\nCheck Interval: **%ds**",
			thresholdMbps, intervalSeconds,
		)
	}

	embed := discordEmbed{
		Title:       "🚀 Monitor Initialized",
		Description: description,
		Color:       3447003,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	payload := discordWebhookPayload{
		Username: "Network Monitor",
		Embeds:   []discordEmbed{embed},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal init discord payload: %w", err)
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create init http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send init discord notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received non-2xx status code from discord on init: %d %s - %s", resp.StatusCode, resp.Status, string(bodyBytes))
	}

	log.Println("Successfully sent initialization notification to Discord.")
	return nil
}

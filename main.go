package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

type GoogIPRanges struct {
	SyncToken    string         `json:"syncToken"`
	CreationTime string         `json:"creationTime"`
	Prefixes     []PrefixRecord `json:"prefixes"`
}

type PrefixRecord struct {
	IPv4Prefix string `json:"ipv4Prefix,omitempty"`
	IPv6Prefix string `json:"ipv6Prefix,omitempty"`
}

func main() {
	url := "https://www.gstatic.com/ipranges/goog.json"

	data, err := fetchGoogIPRanges(url)
	if err != nil {
		log.Fatalf("ошибка при загрузке данных: %v", err)
	}

	file, err := os.Create("routes.txt")
	if err != nil {
		log.Fatalf("не удалось создать файл: %v", err)
	}

	defer func() {
		if cerr := file.Close(); cerr != nil {
			log.Printf("Не удалось корректно закрыть файл: %v", cerr)
		}
	}()

	for _, pr := range data.Prefixes {
		if pr.IPv4Prefix == "" {
			continue
		}

		routeCmd, err := buildRouteCommand(pr.IPv4Prefix)
		if err != nil {
			log.Printf("Не удалось обработать %s: %v", pr.IPv4Prefix, err)
			continue
		}

		fmt.Println(routeCmd)

		_, writeErr := file.WriteString(routeCmd + "\n")
		if writeErr != nil {
			log.Printf("не удалось записать строку: %v", writeErr)
		}
	}
}

func fetchGoogIPRanges(url string) (*GoogIPRanges, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть URL %s: %w", url, err)
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("Не удалось корректно закрыть resp.Body: %v", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("получен некорректный HTTP-статус %d", resp.StatusCode)
	}

	var data GoogIPRanges
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("не удалось декодировать JSON: %w", err)
	}
	return &data, nil
}

func buildRouteCommand(cidr string) (string, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("неверный CIDR %s: %w", cidr, err)
	}

	network := ipNet.IP

	netmask := ipMaskToString(ipNet.Mask)

	return fmt.Sprintf("route ADD %s MASK %s 0.0.0.0",
		network.String(),
		netmask,
	), nil
}

func ipMaskToString(mask net.IPMask) string {
	ip := net.IP(mask).String()
	return ip
}

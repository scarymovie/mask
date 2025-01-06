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

func readExtras(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл %s: %w", filename, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Printf("Не удалось корректно закрыть файл %s: %v", filename, cerr)
		}
	}()

	var extras []string
	if err := json.NewDecoder(f).Decode(&extras); err != nil {
		return nil, fmt.Errorf("не удалось декодировать extras.json: %w", err)
	}
	return extras, nil
}

func main() {
	url := "https://www.gstatic.com/ipranges/goog.json"
	data, err := fetchGoogIPRanges(url)
	if err != nil {
		log.Fatalf("ошибка при загрузке данных: %v", err)
	}

	extras, err := readExtras("extras.json")
	if err != nil {
		log.Printf("Внимание: не удалось прочитать extras.json: %v", err)
		extras = []string{}
	}

	combinedPrefixes := make([]string, 0, len(data.Prefixes)+len(extras))
	for _, pr := range data.Prefixes {
		if pr.IPv4Prefix != "" {
			combinedPrefixes = append(combinedPrefixes, pr.IPv4Prefix)
		}
	}
	combinedPrefixes = append(combinedPrefixes, extras...)

	outFile, err := os.Create("routes.txt")
	if err != nil {
		log.Fatalf("не удалось создать файл routes.txt: %v", err)
	}
	defer func() {
		if cerr := outFile.Close(); cerr != nil {
			log.Printf("Не удалось корректно закрыть файл routes.txt: %v", cerr)
		}
	}()

	for _, cidr := range combinedPrefixes {
		routeCmd, err := buildRouteCommand(cidr)
		if err != nil {
			log.Printf("Не удалось обработать %s: %v", cidr, err)
			continue
		}

		// Выводим в консоль
		fmt.Println(routeCmd)
		// Пишем в файл
		if _, err := outFile.WriteString(routeCmd + "\n"); err != nil {
			log.Printf("Не удалось записать строку в файл: %v", err)
		}
	}
}

func fetchGoogIPRanges(url string) (*GoogIPRanges, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть URL %s: %w", url, err)
	}
	// Корректная обработка ошибки при закрытии тела ответа
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
	return net.IP(mask).String()
}

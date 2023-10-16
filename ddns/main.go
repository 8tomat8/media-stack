package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cloudflare/cloudflare-go"
	tld "github.com/jpillora/go-tld"
	"github.com/pkg/errors"
)

var (
	domain  = os.Getenv("DOMAIN")
	cfToken = os.Getenv("CLOUDFLARE_API_TOKEN")
	lastIP  = ""
)

func main() {
	api, err := cloudflare.NewWithAPIToken(cfToken)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to create cloudflare api client"))
	}

	u, err := tld.Parse(domain)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to parse url"))
	}
	mainDomain := u.Domain + "." + u.TLD
	targetDomain := u.Subdomain + "." + mainDomain

	zoneID, err := api.ZoneIDByName(mainDomain)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to get zone id"))
	}

	// Most API calls require a Context
	ctx := context.Background()

	recs, _, err := api.ListDNSRecords(context.Background(), cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{Name: targetDomain})
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to list dns records"))
	}
	if len(recs) == 0 {
		log.Fatal("no dns record found, please create it first")
	}

	currentRecord := recs[0]

	for {
		func() {
			ip, err := getMyIP()
			if err != nil {
				log.Println(err)
				return
			}
			if ip == lastIP {
				log.Println("ip not changed")
				return
			}
			log.Println("ip changed to", ip)

			_, err = api.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.UpdateDNSRecordParams{
				ID:      currentRecord.ID,
				Type:    "A",
				Name:    targetDomain,
				Content: ip,
				Proxied: currentRecord.Proxied,
				TTL:     currentRecord.TTL,
			})
			if err != nil {
				log.Println(errors.Wrap(err, "failed to update dns record"))
				return
			}
			log.Println("dns record updated")
			lastIP = ip
		}()

		time.Sleep(1 * time.Minute)
	}
}

func getMyIP() (string, error) {
	resp, err := http.Get("https://api.ipify.org?format=text")
	if err != nil {
		return "", errors.Wrap(err, "failed to call ipify")
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read ipify response")
	}
	return string(ip), nil
}

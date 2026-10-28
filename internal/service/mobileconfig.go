package service

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/google/uuid"
	"github.com/safari3308/adguard-portal/internal/domain"
)

type MobileConfigData struct {
	DeviceName  string
	ClientToken string
	DOHDomain   string
	PayloadUUID string
	DNSUUID     string
}

const mobileConfigTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadDisplayName</key>
    <string>AdGuard DoH - {{ .DeviceName }}</string>
    <key>PayloadDescription</key>
    <string>Encrypted DNS over HTTPS configuration for {{ .DeviceName }}</string>
    <key>PayloadIdentifier</key>
    <string>com.adguardportal.doh.{{ .ClientToken }}</string>
    <key>PayloadOrganization</key>
    <string>AdGuard Portal</string>
    <key>PayloadScope</key>
    <string>System</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>{{ .PayloadUUID }}</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
    <key>PayloadContent</key>
    <array>
        <dict>
            <key>PayloadType</key>
            <string>com.apple.dnsSettings.managed</string>
            <key>PayloadScope</key>
            <string>System</string>
            <key>PayloadDisplayName</key>
            <string>AdGuard DoH - {{ .DeviceName }}</string>
            <key>PayloadDescription</key>
            <string>Encrypted DNS over HTTPS configuration for {{ .DeviceName }}</string>
            <key>PayloadIdentifier</key>
            <string>com.adguardportal.doh.{{ .ClientToken }}.dnsSettings</string>
            <key>PayloadUUID</key>
            <string>{{ .DNSUUID }}</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
            <key>DNSSettings</key>
            <dict>
                <key>DNSProtocol</key>
                <string>HTTPS</string>
                <key>ServerURL</key>
                <string>https://{{ .DOHDomain }}/dns-query/{{ .ClientToken }}</string>
            </dict>
            <key>ProhibitDisablement</key>
            <false/>
        </dict>
    </array>
</dict>
</plist>`

var tmpl = template.Must(template.New("mobileconfig").Parse(mobileConfigTemplate))

func GenerateMobileConfig(device *domain.Device, dohDomain string) ([]byte, error) {
	data := MobileConfigData{
		DeviceName:  device.Name,
		ClientToken: device.ClientToken,
		DOHDomain:   dohDomain,
		PayloadUUID: uuid.NewString(),
		DNSUUID:     uuid.NewString(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute mobileconfig template: %w", err)
	}
	return buf.Bytes(), nil
}

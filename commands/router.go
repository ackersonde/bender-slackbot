package commands

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var logger = log.New(os.Stderr, "ASUS_Router: ", log.Lshortfile|log.LstdFlags)

func accessInsecureHTTPClient() *http.Client {
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	return &http.Client{Transport: transport}
}

func loginToRouter(client *http.Client) string {
	// Login to Router
	body := strings.NewReader(`group_id=&action_mode=&action_script=&action_wait=5&current_page=Main_Login.asp&next_page=index.asp&login_authorization=YWRtaW46UnVtcDEzU3RpMXoh`)
	req, err := http.NewRequest("POST", "https://192.168.1.1:8443/login.cgi", body)
	if err != nil {
		logger.Printf("Err Login Request: %+v\n", err)
	}
	resp, err2 := client.Do(req)
	if err2 != nil {
		logger.Printf("Err Login Do: %+v\n", err2)
	}
	defer resp.Body.Close()

	// get Cookie variable value from [asus_token=xyz123; HttpOnly;]
	return strings.TrimLeft(strings.TrimRight(resp.Cookies()[0].Raw, " HttpOnly;"), "=")
}

// ResetMediaServer is now commented
func ResetMediaServer() bool {
	result := false
	client := accessInsecureHTTPClient()

	if runningFritzboxTunnel() {
		asusToken := loginToRouter(client)

		// Restart Media Server
		body2 := strings.NewReader(`preferred_lang=EN&firmver=3.0.0.4&current_page=mediaserver.asp&next_page=mediaserver.asp&flag=nodetect&action_mode=apply&action_script=restart_media&action_wait=5&daapd_enable=0&dms_enable=1&dms_dir_x=%3C%2Fmnt%2FTOSHIBA_EXT%2FDLNA&dms_dir_type_x=%3CV&dms_dir_manual=1&daapd_friendly_name=RT-AC88U-D7F8&dms_friendly_name=EntertainME&dms_rebuild=0&dms_web=1&dms_dir_manual_x=1&type_A_audio=on&type_P_image=on&type_V_video=on`)
		req2, err3 := http.NewRequest("POST", "https://192.168.1.1:8443/start_apply.htm", body2)
		if err3 != nil {
			logger.Printf("Err Restart Request: %+v\n", err3)
		}
		req2.Header.Set("Cookie", asusToken)
		req2.Header.Set("Referer", "https://192.168.1.1:8443/mediaserver.asp")
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp2, err4 := client.Do(req2)
		if err4 != nil {
			logger.Printf("Err Restart Do: %+v\n", err4)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode == 200 { // OK
			bodyBytes, _ := ioutil.ReadAll(resp2.Body)
			bodyString := string(bodyBytes)
			if strings.Contains(bodyString, "no_changes_and_no_committing()") {
				result = true
			}
		}
	}

	return result
}

// ToggleWLANPower is now commented
func ToggleWLANPower(powerFlag string) bool {
	result := false
	client := accessInsecureHTTPClient()

	if runningFritzboxTunnel() {
		asusToken := loginToRouter(client)

		// wl_unit is "tab" of WLAN Band selector 0:2.4GHz, 1:5.0GHz
		// wl_radio is 0|1 toggle off|on

		// Disable 2.4G Wireless Radio Transmitter
		body := strings.NewReader(`wl_unit=` + powerFlag + `&wl_radio=1&productid=RT-AC88U&wl_nmode_x=0&wl_gmode_protection_x=&current_page=Advanced_WAdvanced_Content.asp&next_page=Advanced_WAdvanced_Content.asp&group_id=&modified=0&first_time=&action_mode=apply_new&action_script=restart_wireless&action_wait=10&preferred_lang=EN&firmver=3.0.0.4&wl_subunit=-1&wl_amsdu=auto&wl_TxPower=&wl1_80211h_orig=&acs_dfs=1&w_Setting=1&wl_sched=010500%3C120500%3C230500%3C340500%3C450500%3C560500%3C600500&wl_txpower=100&wl_timesched=1&wl_ap_isolate=0&wl_rate=0&wl_user_rssi=0&wl_igs=0&wl_mrate_x=0&wl_rateset=default&wl_plcphdr=long&wl_ampdu_rts=1&wl_rts=2347&wl_dtim=3&wl_bcn=100&wl_frameburst=on&wl_PktAggregate=0&wl_wme_apsd=on&wl_DLSCapable=0&wl_btc_mode=0&usb_usb3=1&wl_ampdu_mpdu=0&wl_turbo_qam=2&wl1_80211h=0&wl_atf=1&wl_mumimo=0&wl_txbf=1&wl_itxbf=1`)
		req, err := http.NewRequest("POST", "https://192.168.1.1:8443/start_apply.htm", body)
		if err != nil {
			logger.Printf("Err Toggle 2.4G Request: %+v\n", err)
		}
		req.Header.Set("Cookie", asusToken)
		req.Header.Set("Referer", "https://192.168.1.1:8443/Advanced_WAdvanced_Content.asp")

		resp, err2 := client.Do(req)
		if err2 != nil {
			logger.Printf("Err Toggle 2.4G Do: %+v\n", err2)
		}
		defer resp.Body.Close()
		if resp.StatusCode == 200 { // OK
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			bodyString := string(bodyBytes)
			if strings.Contains(bodyString, "done_committing()") {
				result = true
			}
		}

		// Disable 5G Wireless Radio Transmitter
		body2 := strings.NewReader(`wl_unit=` + powerFlag + `&wl_radio=0&productid=RT-AC88U&wl_nmode_x=0&wl_gmode_protection_x=&current_page=Advanced_WAdvanced_Content.asp&next_page=Advanced_WAdvanced_Content.asp&group_id=&modified=0&first_time=&action_mode=apply_new&action_script=restart_wireless&action_wait=10&preferred_lang=EN&firmver=3.0.0.4&wl_subunit=-1&wl_amsdu=auto&wl_TxPower=&wl1_80211h_orig=&acs_dfs=1&w_Setting=1&wl_sched=010500%3C120500%3C230500%3C340500%3C450500%3C560500%3C600500&wl_txpower=100&wl_timesched=1&wl_ap_isolate=0&wl_rate=0&wl_user_rssi=0&wl_igs=0&wl_mrate_x=0&wl_rateset=default&wl_plcphdr=long&wl_ampdu_rts=1&wl_rts=2347&wl_dtim=3&wl_bcn=100&wl_frameburst=on&wl_PktAggregate=0&wl_wme_apsd=on&wl_DLSCapable=0&wl_ampdu_mpdu=0&wl_turbo_qam=2&wl1_80211h=0&wl_atf=1&wl_mumimo=0&wl_txbf=1&wl_itxbf=1`)
		req2, err3 := http.NewRequest("POST", "https://192.168.1.1:8443/start_apply.htm", body2)
		if err3 != nil {
			logger.Printf("Err Toggle 5G Request: %+v\n", err3)
		}
		req2.Header.Set("Cookie", asusToken)
		req2.Header.Set("Referer", "https://192.168.1.1:8443/Advanced_WAdvanced_Content.asp")

		resp2, err4 := client.Do(req2)
		if err4 != nil {
			logger.Printf("Err Toggle 5G Do: %+v\n", err4)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode == 200 { // OK
			bodyBytes, _ := ioutil.ReadAll(resp2.Body)
			bodyString := string(bodyBytes)
			if strings.Contains(bodyString, "done_committing()") {
				result = true
			}
		}
	}

	return result
}

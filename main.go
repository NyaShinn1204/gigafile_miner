package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"bufio"

	"github.com/schollz/progressbar/v3"
)

var lower = []rune("abcdefghijklmnopqrstuvwxyz")
var upper = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
var num = []rune("0123456789")

// ランダムな文字列を生成
func randStr(length int) string {
	charset := append(lower, upper...)
	charset = append(charset, num...)
	rand.Seed(time.Now().UnixNano())
	randomStr := make([]rune, length)
	for i := range randomStr {
		randomStr[i] = charset[rand.Intn(len(charset))]
	}
	return string(randomStr)
}

// プロキシリストをファイルから取得
func getProxies() ([]string, error) {
	file, err := os.Open("http_proxie.txt")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var proxies []string
	for scanner.Scan() {
		proxies = append(proxies, scanner.Text())
	}

	if len(proxies) == 0 {
		return nil, fmt.Errorf("no proxies found")
	}

	return proxies, nil
}

func getFilenameFromCD(cd string) string {
	if cd == "" {
		return ""
	}
	var filename string
	if strings.Contains(cd, "filename*=") {
		// Handle UTF-8 encoded filename
		parts := strings.Split(cd, "filename*=")
		if len(parts) > 1 {
			encodedFilename := strings.Split(parts[1], "''")[1]
			decodedFilename, _ := url.QueryUnescape(encodedFilename)
			filename = decodedFilename
		}
	} else if strings.Contains(cd, "filename=") {
		// Handle regular filename
		parts := strings.Split(cd, "filename=")
		if len(parts) > 1 {
			filename = strings.Trim(strings.Trim(parts[1], "\""), " ")
		}
	}
	return filename
}

func downloadFileWithProgress(url string, folderName string, filename string, cookies []*http.Cookie) error {
	// Get File size
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	totalSize := response.ContentLength

	filePath := filepath.Join(folderName, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	bar := progressbar.NewOptions64(totalSize,
		progressbar.OptionSetDescription(filename),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionSetWidth(15),
	)

	_, err = io.Copy(io.MultiWriter(file, bar), response.Body)
	return err
}

func download_program_main(worker_id int, folderName string, url string, dlkey string) (error, string) {
	// URL to parse
	parts := strings.Split(url, "/")
	domain := parts[2]
	fid := parts[3]

	dlkey = ""
	if err := os.MkdirAll(folderName, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create folder: %v", err), ""
	}

	// First request to get cookies
	client := &http.Client{}
	headReq, _ := http.NewRequest("HEAD", url, nil)
	headResp, err := client.Do(headReq)
	if err != nil {
		fmt.Println("Error:", err)
		return err, ""
	}
	defer headResp.Body.Close()

	// File Download
	dlkeyParam := ""
	if dlkey != "" {
		dlkeyParam = fmt.Sprintf("&dlkey=%s", dlkey)
	}
	nurl := fmt.Sprintf("https://%s/download.php?file=%s%s", domain, fid, dlkeyParam)

	downloadReq, _ := http.NewRequest("GET", nurl, nil)
	for _, cookie := range headResp.Cookies() {
		downloadReq.AddCookie(cookie)
	}
	downloadResp, err := client.Do(downloadReq)
	if err != nil {
		fmt.Println("Error:", err)
		return err, ""
	}
	defer downloadResp.Body.Close()

	filename := getFilenameFromCD(downloadResp.Header.Get("Content-Disposition"))

	if downloadResp.StatusCode == 200 && filename != "" {
		err = downloadFileWithProgress(nurl, folderName, filename, headResp.Cookies())
		if err != nil {
			fmt.Println("Error downloading file:", err)
			return err, ""
		}
	} else {
		fmt.Println("failed, trying different api point")
		nurl = fmt.Sprintf("https://%s/dl_zip.php?file=%s%s", domain, fid, dlkeyParam)
		downloadReq, _ = http.NewRequest("GET", nurl, nil)
		for _, cookie := range headResp.Cookies() {
			downloadReq.AddCookie(cookie)
		}
		downloadResp, err = client.Do(downloadReq)
		if err != nil {
			fmt.Println("Error:", err)
			return err, ""
		}
		defer downloadResp.Body.Close()
		filename = getFilenameFromCD(downloadResp.Header.Get("Content-Disposition"))

		if downloadResp.StatusCode == 200 && filename != "" {
			err = downloadFileWithProgress(nurl, folderName, filename, headResp.Cookies())
			if err != nil {
				fmt.Println("Error downloading file:", err)
				return err, ""
			}
		} else {
			fmt.Println("failed to download file")
			return fmt.Errorf("unknow error maybe password require: %v", downloadResp.StatusCode), ""
		}
	}

	return nil, filename
}

// 各リクエストを処理
func worker(id int, baseURL string, proxies []string, wg *sync.WaitGroup) {
	defer wg.Done()
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for {
		randNum := rand.Intn(2) + 4
		randomString := randStr(randNum)
		newURL := baseURL + randomString
		//newURL := "https://39.gigafile.nu/0817-dc18046c23a5148f784ef24942ab627b1"
		fmt.Printf("\r[Worker %d] [+] [ %s ] mining...", id, randomString)

		proxyAddr := proxies[rand.Intn(len(proxies))]
		proxyURL, err := url.Parse("http://" + proxyAddr)
		if err != nil {
			fmt.Printf("[Worker %d] Error parsing proxy URL: %v\n", id, err)
			continue
		}

		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}

		resp, err := client.Get(newURL)
		if err != nil {
			//fmt.Printf("\n[Worker %d] Request error: %v\n", id, err)
			continue
		}
		defer resp.Body.Close()

		// マッチするURLを発見した場合
		matchedURL := regexp.MustCompile(`^https://\d+\.gigafile\.nu/([a-z0-9-]+)$`).FindStringSubmatch(resp.Request.URL.String())
		if len(matchedURL) > 1 {
			folderName := "download-file/" + matchedURL[1]
			//folderName := "download-file/" + matchedURL[1]
			fmt.Printf("\r[Worker %d] [*] [ %s ] discover!! --> \033[33m%s\033[0m\n", id, randomString, resp.Request.URL.String())
			//if err := downloadFile(resp.Request.URL.String(), folderName, client); err != nil {
			//	fmt.Printf("[Worker %d] Error downloading file: %v\n", id, err)
			//}
			file, err := os.OpenFile("link-list.txt", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
			if err != nil {
				//エラー処理
				log.Fatal(err)
			}
			defer file.Close()
			err, filename := download_program_main(id, folderName, resp.Request.URL.String(), "")
			if err != nil && filename == "" {
				fmt.Printf("[Worker %d] Error downloading file: %v\n", id, err)
				fmt.Fprintln(file, fmt.Sprintf("X %s", resp.Request.URL.String())) //書き込み
			} else {
				fmt.Printf("[Worker %d] Download Success: %v\n", id, filename)
				fmt.Fprintln(file, fmt.Sprintf("O %s", resp.Request.URL.String())) //書き込み
			}
		}
	}
}

func main() {
	fmt.Println("⠀⠀⠀⠀⠀     ⠀⠀   ⣠⣄⠀")
	fmt.Println("	⠀ ⠀⠀⠀⠀⢀⣿⣽⡷  　\033[0m\033[36mGigaFile Miner\033[0m")
	fmt.Println("	⠀⢀⣤⣄⢀⣴⠟⠁⠀⠀    \033[0mSuper Simple GigaFIle Miner By NyaShinn1204\033[0m")
	fmt.Println("	⢠⡟⢁⣽⢿⣅⠀⠀⠀⠀　  \033[0mRespect By Rody\033[0m")
	fmt.Println("	⠸⣇⣀⣁⣴⠟⠀⠀⠀⠀    \033[0mhttps://github.com/NyaShinn1204/gigafile_miner\033[0m")
	fmt.Println("	⠀⠈⠉⠉\n")

	proxies, err := getProxies()
	if err != nil {
		fmt.Println("Error loading proxies:", err)
		return
	}

	baseURL := "https://xgf.nu/"
	numWorkers := 25 // 並行して動作させるワーカーの数
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i, baseURL, proxies, &wg)
	}

	wg.Wait() // すべてのワーカーが終了するのを待機
}

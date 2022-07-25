package main

import (
	"fmt"
    "context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
//    "github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/sagemakerruntime"
	//"reflect"
	"sync"
	"flag"
	"time"
	"net"
    "net/http"
    "github.com/pkg/profile"
 //   "runtime"

	"golang.org/x/net/http2"
)

import _ "net/http/pprof"


func NewHTTPClientWithSettings(httpSettings HTTPClientSettings) (*http.Client, error) {
    var client http.Client
    tr := &http.Transport{
        ResponseHeaderTimeout: httpSettings.ResponseHeader,
        Proxy:                 http.ProxyFromEnvironment,
        DialContext: (&net.Dialer{
            KeepAlive: httpSettings.ConnKeepAlive,
            DualStack: true,
            Timeout:   httpSettings.Connect,
        }).DialContext,
        MaxIdleConns:          httpSettings.MaxAllIdleConns,
        IdleConnTimeout:       httpSettings.IdleConn,
        TLSHandshakeTimeout:   httpSettings.TLSHandshake,
        MaxIdleConnsPerHost:   httpSettings.MaxHostIdleConns,
        ExpectContinueTimeout: httpSettings.ExpectContinue,
    }

    // So client makes HTTP/2 requests
    err := http2.ConfigureTransport(tr)
    if err != nil {
        return &client, err
    }

    return &http.Client{
        Transport: tr,
    }, nil
}

type HTTPClientSettings struct {
		Connect          time.Duration
		ConnKeepAlive    time.Duration
		ExpectContinue   time.Duration
		IdleConn         time.Duration
		MaxAllIdleConns  int
		MaxHostIdleConns int
		ResponseHeader   time.Duration
		TLSHandshake     time.Duration
	}


func main() {
    

    
     go func() {
        
        http.ListenAndServe(":1234",nil)
    }()
    
    /*go func() {
        for i := 0; i < 8; i++ {
            fmt.Printf("Dumping threads for same process at %d iteration.",i)
            buf := make([]byte, 1<<16)
            runtime.Stack(buf, true)
            fmt.Printf("%s", buf)
            time.Sleep(100 * time.Millisecond)
        }
    }()*/

    defer profile.Start().Stop() 

	httpClient, err := NewHTTPClientWithSettings(HTTPClientSettings{
		Connect:          5 * time.Second,
		ExpectContinue:   1 * time.Second,
		IdleConn:         90 * time.Second,
		ConnKeepAlive:    90 * time.Second,
		MaxAllIdleConns:  100,
		MaxHostIdleConns: 100,
		ResponseHeader:   5 * time.Second,
		TLSHandshake:     50 * time.Second,
	})
	if err != nil {
		fmt.Println("Got an error creating custom HTTP client:")
		fmt.Println(err)
		return
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: credentials.NewSharedCredentials("", "default"),
		HTTPClient: httpClient,
        //Retryer: client.NoOpRetryer{},
	})
	if err != nil {
		fmt.Println(err) // Does not fail here
	}

	_, err = sess.Config.Credentials.Get()
	if err != nil {
		fmt.Println(err) // Does not fail here
	}

	svc := sagemakerruntime.New(sess)

    var wg sync.WaitGroup
    routines := flag.Int("runs", 25, "routines running")
    flag.Parse()
	ctx := context.Background()

    wg.Add(*routines)

	for i := 0; i < *routines; i++ {
		go invoke(ctx, svc, &wg) 
	}

    wg.Wait()
    
    

}

func invoke(ctx context.Context, svc *sagemakerruntime.SageMakerRuntime, wg *sync.WaitGroup ) (duration time.Duration) {
    //defer profile.Start(profile.ProfilePath(".")).Stop()
       
    
	defer wg.Done()

	body := "2,0,0,1,58400,14247.76686667085,72647.76686667085,7,29,0,11,41,108,0,1,750,3000,0,2,2014,0,0,1,0,0,1,0,0,0,1,0,0,0,0,1,0,0,1,0,1,0,0,0,0,0,0,0,1"
	params := sagemakerruntime.InvokeEndpointInput{}
	params.SetAccept("text/csv")
	params.SetContentType("text/csv")
	params.SetBody([]byte(body))
	params.SetEndpointName("fraud-detect-xgb-endpoint")
	start := time.Now()
    trace := &RequestTrace{}
    _, err := svc.InvokeEndpointWithContext(ctx, &params, trace.TraceRequest)
    fmt.Println(trace)
	//_, err := svc.InvokeEndpoint(&params)
	duration = time.Since(start)
	if err != nil {
		fmt.Println(err) // errors here:
		// InvalidSignatureException: Credential should be scoped to correct service: 'sagemaker'.
		// status code: 403, request id: 123-abc....
	}

	fmt.Println("Time taken to invokeendpoint was %d",duration)
	
	return duration


}
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var sites = []string{
	"http://ifconfig.me/ip",
	"https://ifconfig.co/",
	"https://api.ipify.org/",
	"http://ipv4bot.whatismyipaddress.com/",
}

type Ana struct {
	Session *session.Session
}

func newAna(awsAccessKeyID, awsSecretAccessKey, awsRegion string) *Ana {
	creds := credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, "")
	sess := session.New(&aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: creds,
	})
	return &Ana{Session: sess}
}

func (ana *Ana) open(cidr, securityGroupID string) error {
	svc := ec2.New(ana.Session)

	params := &ec2.AuthorizeSecurityGroupIngressInput{
		CidrIp:     aws.String(cidr),
		DryRun:     aws.Bool(false),
		FromPort:   aws.Int64(22),
		ToPort:     aws.Int64(22),
		GroupId:    aws.String(securityGroupID),
		IpProtocol: aws.String("tcp"),
	}
	_, err := svc.AuthorizeSecurityGroupIngress(params)

	if err != nil {
		if strings.Contains(err.Error(), "InvalidPermission.Duplicate") {
			fmt.Printf("%s:22 is already opened\n", cidr)
			return nil
		}
		return err
	}
	return nil
}

func (ana *Ana) close(cidr, securityGroupID string) error {
	svc := ec2.New(ana.Session)

	params := &ec2.RevokeSecurityGroupIngressInput{
		CidrIp:     aws.String(cidr),
		DryRun:     aws.Bool(false),
		FromPort:   aws.Int64(22),
		ToPort:     aws.Int64(22),
		GroupId:    aws.String(securityGroupID),
		IpProtocol: aws.String("tcp"),
	}
	_, err := svc.RevokeSecurityGroupIngress(params)

	if err != nil {
		if strings.Contains(err.Error(), "InvalidPermission.NotFound") {
			fmt.Printf("%s:22 is already closed\n", cidr)
			return nil
		}
		return err
	}
	return nil
}

func getIPAddress(ctx context.Context, url string) (string, error) {

	tr := &http.Transport{}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return "", err
	}

	errCh := make(chan error)
	ipCh := make(chan string)
	go func() {
		res, err := client.Do(req)
		if err != nil {
			errCh <- err
			return
		}

		ip, err := ioutil.ReadAll(res.Body)
		if err != nil {
			errCh <- err
			return
		}
		ipCh <- string(ip)
	}()

	select {
	case ip := <-ipCh:
		return string(ip), nil
	case err := <-errCh:
		return "", err
	case <-ctx.Done():
		tr.CancelRequest(req)
		<-errCh
		return "", ctx.Err()
	}
}

func main() {

	securityGroupID := os.Getenv("AWS_SECURITY_GROUP_ID")
	awsAccessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	awsRegion := os.Getenv("AWS_REGION")
	if securityGroupID == "" || awsAccessKeyID == "" || awsSecretAccessKey == "" || awsRegion == "" {
		fmt.Println("set AWS_SECURITY_GROUP_ID, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY and AWS_REGION")
		return
	}

	if len(os.Args) != 2 {
		fmt.Println("args must be 'open' or 'close'")
		return
	}
	op := os.Args[1]

	ipCh := make(chan string)
	errCh := make(chan error)
	findCh := make(chan bool)

	ctx, cancel := context.WithTimeout(context.Background(), 5000*time.Millisecond)
	defer cancel()

	for _, site := range sites {
		go func(siteUrl string) {
			ip, err := getIPAddress(ctx, siteUrl)
			if err != nil {
				errCh <- err
			} else {
				ipCh <- ip
			}
		}(site)
	}

	var myIP string
	go func() {
		for i := 0; i < len(sites); i++ {
			select {
			case ip := <-ipCh:
				trimmedIP := strings.TrimSpace(ip)
				if net.ParseIP(trimmedIP) == nil {
					fmt.Printf("Skipping... ip [%s] invalid\n", ip)
					continue
				}
				myIP = trimmedIP
				findCh <- true
				return
			case err := <-errCh:
				fmt.Printf("Skipping... %s\n", err.Error())
			}
		}
		findCh <- false
	}()

	find := <-findCh
	if find == false || myIP == "" {
		fmt.Println("Could not get ip address")
		return
	}

	var err error
	cidr := fmt.Sprintf("%s/32", myIP)

	fmt.Printf("Trying to %s %s...\n", op, cidr)
	ana := newAna(awsAccessKeyID, awsSecretAccessKey, awsRegion)

	switch op {
	case "open":
		err = ana.open(cidr, securityGroupID)
	case "close":
		err = ana.close(cidr, securityGroupID)
	default:
		fmt.Println("args must be 'open' or 'close'")
		return
	}

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("%s success\n", op)
	}
}

package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var sites = []string{
	"http://ifconfig.me/ip",
	"https://api.ipify.org",
	"http://ipv4bot.whatismyipaddress.com",
}

type Ana struct {
	Session *session.Session
}

func newAna(awsAccessKeyId, awsSecretAccessKey, awsRegion string) *Ana {
	creds := credentials.NewStaticCredentials(awsAccessKeyId, awsSecretAccessKey, "")
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
			fmt.Printf("%s:22 is already opened", cidr)
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

func getIPAddress(url string) (string, error) {

	res, _ := http.Get(url)
	ip, _ := ioutil.ReadAll(res.Body)
	return string(ip), nil
}

func main() {

	securityGroupID := os.Getenv("AWS_SECURITY_GROUP_ID")
	if securityGroupID == "" {
		fmt.Println("set AWS_SECURITY_GROUP_ID")
		return
	}
	awsAccessKeyId := os.Getenv("AWS_ACCESS_KEY_ID")
	if awsAccessKeyId == "" {
		fmt.Println("set AWS_ACCESS_KEY_ID")
		return
	}
	awsSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if awsSecretAccessKey == "" {
		fmt.Println("set AWS_SECRET_ACCESS_KEY")
		return
	}
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		fmt.Println("set AWS_REGION")
		return
	}

	if len(os.Args) != 2 {
		fmt.Println("args must be 'open' or 'close'")
		return
	}
	op := os.Args[1]

	ipCh := make(chan string)

	for _, site := range sites {
		go func(siteUrl string) {
			ip, _ := getIPAddress(site)
			if net.ParseIP(ip) == nil {
				return
			}
			ipCh <- ip
		}(site)
	}

	var myIP string
	select {
	case ip := <-ipCh:
		myIP = ip
	}

	cidr := fmt.Sprintf("%s/32", myIP)
	ana := newAna(awsAccessKeyId, awsSecretAccessKey, awsRegion)

	var err error
	switch op {
	case "open":
		err = ana.open(cidr, securityGroupID)
	case "close":
		err = ana.close(cidr, securityGroupID)
	default:
		return
	}

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("%s success", op)
	}
}

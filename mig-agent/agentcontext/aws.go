// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com [:alm]
package agentcontext

import (
	"fmt"
	"io/ioutil"
	"mig.ninja/mig"
	"net"
	"net/http"
	"time"
)

// Various agent functions that are specific to an agent that is running
// in Amazon Web Services

const AWSMETAIP string = "169.254.169.254"
const AWSMETAPORT int = 80

// The maximum number of bytes we will fetch in a response from the metadata
// service
const FETCHBODYMAX int64 = 10240

func addAWSMetadata(orig_ctx AgentContext) (ctx AgentContext, err error) {
	ctx = orig_ctx

	// First, check and see if we have access to a valid metadata service
	buf, err := awsFetchMeta("")
	if err != nil || buf == "" {
		logChan <- mig.Log{Desc: "AWS metadata service not found, skipping fetch"}.Debug()
		return ctx, nil
	}

	// Attempt to fetch metadata; if any error occurs we just revert to the
	// previous context
	logChan <- mig.Log{Desc: "Attempting to retrieve AWS instance metadata"}.Debug()
	flist := []func(AgentContext) (AgentContext, error){
		addAWSInstanceID,
		addAWSLocalIPV4,
		addAWSAMIID,
		addAWSInstanceType,
	}
	for i := range flist {
		ctx, err = flist[i](ctx)
		if err != nil {
			logChan <- mig.Log{Desc: fmt.Sprintf("Error during metadata fetch: %v", err)}.Debug()
			return orig_ctx, nil
		}
	}
	logChan <- mig.Log{Desc: "AWS metadata fetch successful"}.Debug()
	return ctx, nil
}

func addAWSInstanceID(orig_ctx AgentContext) (ctx AgentContext, err error) {
	ctx = orig_ctx
	var res string
	res, err = awsFetchMeta("instance-id")
	if err != nil {
		return
	}
	ctx.AWS.InstanceID = res
	return
}

func addAWSLocalIPV4(orig_ctx AgentContext) (ctx AgentContext, err error) {
	ctx = orig_ctx
	var res string
	res, err = awsFetchMeta("local-ipv4")
	if err != nil {
		return
	}
	ctx.AWS.LocalIPV4 = res
	return
}

func addAWSAMIID(orig_ctx AgentContext) (ctx AgentContext, err error) {
	ctx = orig_ctx
	var res string
	res, err = awsFetchMeta("ami-id")
	if err != nil {
		return
	}
	ctx.AWS.AMIID = res
	return
}

func addAWSInstanceType(orig_ctx AgentContext) (ctx AgentContext, err error) {
	ctx = orig_ctx
	var res string
	res, err = awsFetchMeta("instance-type")
	if err != nil {
		return
	}
	ctx.AWS.InstanceType = res
	return
}

func awsFetchMeta(endpoint string) (result string, err error) {
	tr := &http.Transport{
		Dial: (&net.Dialer{Timeout: time.Second}).Dial,
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(awsMetaURL() + endpoint)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("invalid HTTP response code returned by metadata service: %v",
			resp.StatusCode)
		return
	}
	if resp.ContentLength == -1 || resp.ContentLength > FETCHBODYMAX {
		err = fmt.Errorf("invalid content length in response body")
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	result = string(body)
	return
}

func awsMetaURL() string {
	return fmt.Sprintf("http://%v:%v/latest/meta-data/", AWSMETAIP, AWSMETAPORT)
}

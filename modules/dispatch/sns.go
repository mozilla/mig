// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package dispatch /* import "mig.ninja/mig/modules/dispatch" */

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

var snsARN string
var snsService *sns.SNS
var awsSession *session.Session

func initSNS(cfg config) error {
	awsSession = session.Must(session.NewSession())

	meta := ec2metadata.New(awsSession)
	instancedoc, err := meta.GetInstanceIdentityDocument()
	if err != nil {
		return err
	}

	// Build an ARN we will use to publish
	snsARN = "arn:aws:sns:" + instancedoc.Region + ":" +
		instancedoc.AccountID + ":" + cfg.Dispatch.SNSTopic

	snsService = sns.New(awsSession, &aws.Config{
		Region: &instancedoc.Region,
	})
	return nil
}

func dispatchSNS(msg []byte) error {
	params := &sns.PublishInput{
		Message:  aws.String(string(msg)),
		TopicArn: aws.String(snsARN),
	}
	_, err := snsService.Publish(params)
	return err
}

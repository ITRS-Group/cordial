package mq

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginsSchemaFieldsRoundTrip(t *testing.T) {
	zero := 0
	statsCalcTimeout := 30
	falseValue := false
	trueValue := true
	format := "SummaryOnly"

	channel := MQChannelPlugin{
		QueueManager: common.NewValue("QM1"),
		Connection: &MQConnection{
			MQServer: common.NewSingleLineString("mq.example.test"),
			SSL: &MQSSL{
				KeyRepository: common.NewSingleLineString("/keys/repository"),
				CipherSpec:    common.NewSingleLineString("TLS_AES_256_GCM_SHA384"),
				MQSender:      common.NewValue("SENDER"),
			},
			SecurityExitConfig: &MQSecurityExitConfig{
				SecurityExit:         common.NewSingleLineString("/mq/security-exit"),
				RemoteUserIdentifier: common.NewSingleLineString("remote-user"),
				RemotePassword:       &common.Value{ExtPwd: "mq/remote-password"},
			},
		},
		Channels:      []MQChannel{{Matches: common.NewSingleLineString(".*")}},
		Columns:       []string{"name"},
		DisplayFormat: &format,
		Debug: &MQChannelDebug{
			SampleTimeout: &zero,
			Output:        &MQChannelDebugOutput{Setup: &falseValue, Summary: &trueValue},
		},
	}
	assertMQRoundTrip(t, &channel, []string{
		"<queueManager><data>QM1</data></queueManager>",
		"<ssl>", "<keyRepository>", "<cipherSpec>", "<securityExitConfig>",
		"<remotePassword><extPwd>mq/remote-password</extPwd></remotePassword>",
		"<displayFormat>SummaryOnly</displayFormat>", "<sampleTimeout>0</sampleTimeout>",
		"<setup>false</setup>", "<summary>true</summary>",
	})

	queue := MQQueuePlugin{
		QueueManager:                 common.NewValue("QM2"),
		Queues:                       []MQQueue{{StartsWith: common.NewSingleLineString("APP.")}},
		DisableQueueResetAttributes:  common.NewValue(false),
		DisableQueueBrowseAttributes: common.NewValue(true),
		StagnantMessageAge:           common.NewValue(0),
		Debug: &MQQueueDebug{
			SampleTimeout:           &zero,
			DisableBrowseTimeAdjust: &trueValue,
			BrowseWarnThreshold:     &zero,
			Output:                  &MQQueueDebugOutput{Browsing: &falseValue},
		},
	}
	assertMQRoundTrip(t, &queue, []string{
		"<disableQueueResetAttributes><data>false</data></disableQueueResetAttributes>",
		"<disableQueueBrowseAttributes><data>true</data></disableQueueBrowseAttributes>",
		"<stagnantMessageAge><data>0</data></stagnantMessageAge>",
		"<disableBrowseTimeAdjust>true</disableBrowseTimeAdjust>",
		"<browseWarnThreshold>0</browseWarnThreshold>", "<browsing>false</browsing>",
	})

	qinfo := MQQInfoPlugin{
		Queuemanager:       common.NewValue("QM3"),
		QueueName:          common.NewSingleLineString("APP.QUEUE"),
		HideUnavailable:    common.NewValue(false),
		StagnantMessageAge: common.NewValue(1),
		StatsCalcTimeOut:   &statsCalcTimeout,
		SecurityConnectionAuthenticationConfig: &MQSecurityConnAuth{
			Username: common.NewSingleLineString("mq-user"),
			Password: &common.Value{Var: &common.Var{Var: "mqPassword"}},
		},
		Private: &MQPrivate{Debug: common.NewValue(1)},
	}
	assertMQRoundTrip(t, &qinfo, []string{
		"<queueManagerName><data>QM3</data></queueManagerName>",
		"<queueName><data>APP.QUEUE</data></queueName>",
		"<hideUnavailable><data>false</data></hideUnavailable>",
		"<statsCalcTimeOut>30</statsCalcTimeOut>", "<securityConnectionAuthenticationConfig>",
		"<password><var ref=\"mqPassword\"></var></password>",
		"<private><debug><data>1</data></debug></private>",
	})
}

func assertMQRoundTrip(t *testing.T, value any, elements []string) {
	t.Helper()
	data := testhelpers.RoundTrip(t, value)
	for _, element := range elements {
		testhelpers.Contains(t, data, element)
	}
}

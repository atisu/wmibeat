package beater

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/eskibars/wmibeat/config"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

type Wmibeat struct {
	beatConfig *config.Config
	done       chan struct{}
	period     time.Duration
}

// Creates beater
func New() *Wmibeat {
	return &Wmibeat{
		done: make(chan struct{}),
	}
}

/// *** Beater interface methods ***///

func (bt *Wmibeat) Config(b *beat.Beat) error {

	// Load beater beatConfig
	err := cfgfile.Read(&bt.beatConfig, "")
	if err != nil {
		return fmt.Errorf("Error reading config file: %v", err)
	}

	return nil
}

func (bt *Wmibeat) Setup(b *beat.Beat) error {

	// Setting default period if not set
	if bt.beatConfig.Wmibeat.Period == "" {
		bt.beatConfig.Wmibeat.Period = "1s"
	}

	var err error
	bt.period, err = time.ParseDuration(bt.beatConfig.Wmibeat.Period)
	if err != nil {
		return err
	}

	return nil
}

func (bt *Wmibeat) Run(b *beat.Beat) error {
	logp.Info("wmibeat is running! Hit CTRL-C to stop it.")

	ticker := time.NewTicker(bt.period)
	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		ole.CoInitialize(0)
		wmiscriptObj, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
		if err != nil {
			return err
		}
		wmiqi, err := wmiscriptObj.QueryInterface(ole.IID_IDispatch)
		if err != nil {
			return err
		}
		defer wmiscriptObj.Release()
		serviceObj, err := oleutil.CallMethod(wmiqi, "ConnectServer")
		if err != nil {
			return err
		}
		defer wmiqi.Release()
		service := serviceObj.ToIDispatch()
		defer serviceObj.Clear()

		var allValues common.MapStr
		for _, class := range bt.beatConfig.Wmibeat.Classes {
			if len(class.Fields) > 0 {
				var query bytes.Buffer
				wmiFields := class.Fields
				query.WriteString("SELECT ")
				query.WriteString(strings.Join(wmiFields, ","))
				query.WriteString(" FROM ")
				query.WriteString(class.Class)
				if class.WhereClause != "" {
					query.WriteString(" WHERE ")
					query.WriteString(class.WhereClause)
				}
				logp.Info("Query: " + query.String())
				resultObj, err := oleutil.CallMethod(service, "ExecQuery", query.String())
				if err != nil {
					return err
				}
				result := resultObj.ToIDispatch()
				defer resultObj.Clear()
				countObj, err := oleutil.GetProperty(result, "Count")
				if err != nil {
					return err
				}
				count := int(countObj.Val)
				defer countObj.Clear()

				var classValues interface{} = nil

				if class.ObjectTitle != "" {
					classValues = common.MapStr{}
				} else {
					classValues = []common.MapStr{}
				}
				for i := 0; i < count; i++ {
					rowObj, err := oleutil.CallMethod(result, "ItemIndex", i)
					if err != nil {
						return err
					}
					row := rowObj.ToIDispatch()
					defer rowObj.Clear()
					var rowValues common.MapStr
					var objectTitle = ""
					for _, j := range wmiFields {
						wmiObj, err := oleutil.GetProperty(row, j)

						if err != nil {
							return err
						}
						var objValue = wmiObj.Value()
						if class.ObjectTitle == j {
							objectTitle = objValue.(string)
						}
						rowValues = common.MapStrUnion(rowValues, common.MapStr{j: objValue})
						defer wmiObj.Clear()

					}
					if class.ObjectTitle != "" {
						if objectTitle != "" {
							classValues = common.MapStrUnion(classValues.(common.MapStr), common.MapStr{objectTitle: rowValues})
						} else {
							classValues = common.MapStrUnion(classValues.(common.MapStr), common.MapStr{strconv.Itoa(i): rowValues})
						}
					} else {
						classValues = append(classValues.([]common.MapStr), rowValues)
					}
					rowValues = nil
				}
				allValues = common.MapStrUnion(allValues, common.MapStr{class.Class: classValues})
				classValues = nil

			} else {
				var errorString bytes.Buffer
				errorString.WriteString("No fields defined for class ")
				errorString.WriteString(class.Class)
				errorString.WriteString(".  Skipping")
				logp.Warn(errorString.String())
			}
		}

		for _, namespace := range bt.beatConfig.Wmibeat.Namespaces {
			logp.Info("Namespace: root\\" + namespace.Namespace)
			nsServiceObj, err := oleutil.CallMethod(wmiqi, "ConnectServer", "localhost",
				"root\\"+namespace.Namespace)
			if err != nil {
				logp.Err("Cannot connect to namespace `" + namespace.Namespace + "`, skipping it.")
				continue
			}
			nsService := nsServiceObj.ToIDispatch()
			defer serviceObj.Clear()

			var query bytes.Buffer
			metricNameFields := namespace.MetricNameCombinedFields
			var allWMIFields []string = append(metricNameFields, namespace.MetricValueField)
			query.WriteString("SELECT ")
			query.WriteString(strings.Join(allWMIFields, ","))
			query.WriteString(" FROM ")
			query.WriteString(namespace.Class)
			if namespace.WhereClause != "" {
				query.WriteString(" WHERE ")
				query.WriteString(namespace.WhereClause)
			}
			logp.Info("Query: " + query.String())
			resultObj, err := oleutil.CallMethod(nsService, "ExecQuery", query.String())
			if err != nil {
				logp.Err("Cannot exec query in current namespace, skipping it.")
				continue
			}
			result := resultObj.ToIDispatch()
			defer resultObj.Clear()
			countObj, err := oleutil.GetProperty(result, "Count")
			if err != nil {
				logp.Err("Cannot get `Count` property, skipping current namespace.")
				continue
			}
			count := int(countObj.Val)
			defer countObj.Clear()
			logp.Info("Count: " + strconv.Itoa(count))

			for i := 0; i < count; i++ {
				rowObj, err := oleutil.CallMethod(result, "ItemIndex", i)
				if err != nil {
					logp.Err("Cannot fetch ItemIndex, skipping it.")
					continue
				}
				row := rowObj.ToIDispatch()
				defer rowObj.Clear()

				var objectTitle = namespace.Namespace + "_" + namespace.Class
				var metricValue = ""
				var hasError int = 0
				for _, j := range allWMIFields {
					wmiObj, err := oleutil.GetProperty(row, j)
					if err != nil {
						logp.Err("Cannot get `Count` property for row, skipping it.")
						hasError = 1
						break
					}
					var objValue = wmiObj.Value()
					if j != namespace.MetricValueField {
						objectTitle = fmt.Sprintf("%s_%v", objectTitle, objValue)
					} else {
						metricValue = fmt.Sprintf("%v", objValue)
					}
					defer wmiObj.Clear()
				}
				if hasError == 0 {
					objectTitle = strings.ReplaceAll(strings.ReplaceAll(objectTitle, " ", ""), "#", "")
					logp.Info(objectTitle + " = " + metricValue)
					allValues = common.MapStrUnion(allValues, common.MapStr{objectTitle: metricValue})
				}
			}
		}

		ole.CoUninitialize()

		event := common.MapStr{
			"@timestamp": common.Time(time.Now()),
			"type":       b.Name,
			"wmi":        allValues,
		}
		b.Events.PublishEvent(event)
		logp.Info("Event sent")
	}
}

func (bt *Wmibeat) Cleanup(b *beat.Beat) error {
	return nil
}

func (bt *Wmibeat) Stop() {
	close(bt.done)
}

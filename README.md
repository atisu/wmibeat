# WMIbeat-OHM

This is an extended version of WMIbeat, mainly for supporting OpenHardwareMonitor style metrics from WMI. However, it is more general, it allows connecting to custom namespaces and retrieveing metrics, using a bit different configuration format.

From the section "WMIbeat" the original README is kept, only the following subsection(s) are for the customization.

## Extended configuration options
```YAML
namespaces:
    - namespace: OpenHardwareMonitor
      class: Sensor
      metric_name_combined_fields:
         - Name
         #- Index
         - SensorType
      metric_value_field: Value
      whereclause: SensorType='Load' OR SensorType='Temperature' OR SensorType='Data' OR SensorType='Power'
```
- `namespace` sets the namespace to connect to.
- `class` defines the class within the namespace.
- The values of the fields defined in `metric_name_combined_fields` will be combined and used as the name of the metric (with the addition of the namespace and class and the WMIBeat specific prefix). For example, the values above will produce `<namespace>_<class>_<Name>_<SensorType>`, e.g., `wmibeat_wmi_OpenHardwareMonitor_Sensor_CPUCore1_Load`. Spaces and '#' are removed from metric names.

## Compiling

Please follow the originial description.

As a note, to cross-compile for Windows use e.g., the following:
```
GOOS=windows GOARCH=386 go build -o wmibeat.exe main.go
```

# WMIbeat

Welcome to WMIbeat.  WMIbeat is a [beat](https://github.com/elastic/beats) that allows you to run arbitrary WMI queries
and index the results into [elasticsearch](https://github.com/elastic/elasticsearch) so you can monitor Windows machines.

Ensure that this folder is at the following location:
`${GOPATH}/github.com/eskibars`

## Getting Started with WMIbeat
To get running with WMIbeat, run "go build" and then run wmibeat.exe, as in the below `run` section.
If you don't want to build your own, hop over to the "releases" page to download the latest.

### Configuring
To configure the WMI queries to run, you need to change wmibeat.yml.  Working from the default example:

    classes:
    - class: Win32_OperatingSystem
      fields:
      - FreePhysicalMemory
      - FreeSpaceInPagingFiles
      - FreeVirtualMemory
      - NumberOfProcesses
      - NumberOfUsers
    - class: Win32_PerfFormattedData_PerfDisk_LogicalDisk
      fields:
      - Name
      - FreeMegabytes
      - PercentFreeSpace
      - CurrentDiskQueueLength
      - DiskReadsPerSec
      - DiskWritesPerSec
      - DiskBytesPerSec
      - PercentDiskReadTime
      - PercentDiskWriteTime
      - PercentDiskTime
      whereclause: Name != "_Total"
	  objecttitlecolumn: Name
    - class: Win32_PerfFormattedData_PerfOS_Memory
      fields:
      - CommittedBytes
      - AvailableBytes
      - PercentCommittedBytesInUse

We can configure a set of classes, a set of fields per class, and a whereclause.  If there are multiple results, for any WMI class,
WMIbeat will add the results as arrays.  If you need some help with what classes/fields, you can try [WMI Explorer](https://wmie.codeplex.com/).
Note that many of the more interesting classes are "Perf" classes, which has a special checkbox to see in that tool.

### Run

To run WMIbeat with debugging output enabled, run:

```
./wmibeat -c wmibeat.yml -e -d "*"
```

## Build your own Beat
Beats is open source and has a convenient Beat generator, from which this project is based.
For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).

# awls

Lookup EC2 information in the terminal

## Installation

OSX:
`brew tap danpilch/awls && brew install awls`

## Usage

```bash
Usage: awls <search>

Arguments:
  <search>    EC2 Instance Name search term

Flags:
  -h, --help                      Show context-sensitive help.
  -i, --ip-only                   Output only Private IPs
  -d, --delimiter=" "             IP delimiter
  -f, --filter-type="tag:Name"    EC2 Filter Type (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeInstances.html)
```

### Examples

```bash
# basic tag:Name search
awls searchterm

# fuzzy tag:Name search
awls 'search*term'

# fuzzy tag:cluster search
awls 'dev*ec2*' -f tag:cluster

# Search by instance-id filter and output private IP delimited by ','
awls 'i-0bfxxxxxxx' -f instance-id -i -d,
```

#### Sample Output

```bash
# Search by instance-id filter and output in default table
awls 'i-0bxxxxx' -f instance-id

+------------------------------------------+------------+---------+------------+---------------------+--------------+---------------------+
|                   NAME                   | PRIVATEIP  |  STATE  |     AZ     |     INSTANCEID      | INSTANCETYPE |     LAUNCHTIME      |
+------------------------------------------+------------+---------+------------+---------------------+--------------+---------------------+
| hostname                                 | 10.0.0.1   | running | us-west-2a | i-0bxxxxx           | t2.small     | 2021-01-12 10:18:21 |
+------------------------------------------+------------+---------+------------+---------------------+--------------+---------------------+

# basic tag:Name search and output private IP delimited by ','
awls 'search*term' -i -d,

10.0.0.1,10.0.0.2,10.0.0.3,10.0.0.4
```

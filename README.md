# ana

ana(穴=means hole in Japanese) is a tool to open/close the AWS security group 22 port from your machine. If you'd like to connect via ssh to your EC2 servers from the place such as a cafe and friend's home, the tool would be a very easy method to open ssh port.

# Installation

Use bin/ana_* or build.

# How to use

* settings

```
export AWS_SECURITY_GROUP_ID=<target aws security group id>
export AWS_ACCESS_KEY_ID=<your aws access key id>
export AWS_SECRET_ACCESS_KEY=<your aws secret access key>
export AWS_REGION=<aws region>
```

* run

```
# open. add <your global ip>:22 to the aws security group
ana open
# close. delete <your global ip>:22 from the aws security group
ana close
```

# License

MIT
Development and Testing
-----------------------

ToDo: Create docs for setting up DirectCSI development and testing environment


## Loopback Devices

DirectCSI can automatically provision loopback devices for setups where extra drives are not available. The loopback interface is intended for use with automated testing and continuous integration, and is not recommended for use in regular development or production environments. Some operating systems, such as macOS, place limits on the number of loop devices and can cause DirectCSI to hang while attempting to provision persistent volumes. This issue is particularly noticeable on Kubernetes deployment tools like `kind` or `minikube`, where the deployed infrastructure takes up most if not all of the available loop devices and prevents DirectCSI from provisioning drives entirely.

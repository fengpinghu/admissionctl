FROM centos

RUN yum install -y sssd-client

ADD build/_output/bin/k8sadmissionctl /k8sadmissionctl
ENTRYPOINT ["./k8sadmissionctl"]

# Kubernetes admission control

[readme](https://github.com/fengpinghu/addmissionctl)

This project uses [AdmissionWebhooks](https://kubernetes.io/docs/admin/admission-controllers/) to control kubernetes objects, such as setting securitycontext of containers, labeling pods with Usernames, setting up walltime, validating user rights on kubernetes objects and etc. Together with RBAC, this admission control webhook enables an on prem kubernetes cluster to be shared by users such that host file systems can be controlled and accessed based on user rights, users won't step on each other's toes and many other possiblities...... 

## Notes

Currently the supported kubernetes resource includes:
the native pods, jobs, deployment, as well as the volcano vcjob

- [setup a kubernetes cluster with ldap authentication](https://github.com/krishnapmv/k8s-ldap)
- [plumbing for admission webhooks](https://github.com/morvencao/kube-mutating-webhook-tutorial) 



## Build and Deploy
Make sure to build with cgo enabled.
Make sure user information is available on the admission control container. For example configure it with sssd support.
For other details, please refer to [plumbing for admission webhooks](https://github.com/morvencao/kube-mutating-webhook-tutorial)

## Verify

1. run a container as the submitting user.

Submitting user has this in the jwt
```
{
  ... 
  "groups": [
    "group1",
    "group2"
  ],
  "name": "user1"
}
```

Run busybox and check user id
```
$ kubectl run test -n kube-public --image=busybox -it --rm --restart=Never /bin/sh
If you don't see a command prompt, try pressing enter.
/ $ id
uid=xxx gid=xxx(users) groups=xxx,xxx
```

The uid will match what user1 has on the target system.

2. delete other peoples resource:

```
$ kubectl delete pod -n kube-public apple-app
Error from server (You are not allowed to delete this resource, please contact admin if you have any questions!)
```

## Troubleshooting

1. Check if the user is available on the webhook container.
2. Check if the user is available on the host system 

{{- $name := get . "name"}}
{{- $namespace := get . "namespace"}}
{{- $ownerRefs := get . "ownerRefs"}}

{{- $tolerations := get . "tolerations" |default list }}
{{- $nodeSelector := get . "node-selector" |default dict }}
{{- $tlsCertSecName := get . "tls-cert-sec-name" | default ""}}
{{- $devInfo := get . "dev-info" | default ""}}

apiVersion: apps/v1
kind: Deployment
metadata:
  name: "wg-server-{{$name}}"
  annotations:
    kloudlite.io/account.name: {{ $name }}
  labels:
    kloudlite.io/wg-deployment: "true"
    kloudlite.io/wg-device.name: {{ $name }}
  ownerReferences: {{ $ownerRefs| toJson}}
  namespace: {{ $namespace }}
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      kloudlite.io/pod-type: wireguard-server
      kloudlite.io/device: {{$name}}
  template:
    metadata:
      labels:
        kloudlite.io/pod-type: wireguard-server
        kloudlite.io/device: {{$name}}
    spec:
      nodeSelector: {{$nodeSelector | toJson}}
      tolerations: {{$tolerations | toJson}}
      containers:
      - name: wireguard
        imagePullPolicy: IfNotPresent
        {{- /* image: ghcr.io/kloudlite/platform/apis/wg-restart:v1.0.5-nightly */}}
        image: ghcr.io/linuxserver/wireguard
        securityContext:
          capabilities:
            add:
              - NET_ADMIN
              - SYS_MODULE
          privileged: true
        volumeMounts:
          - name: wg-config
            mountPath: /etc/wireguard/wg0.conf
            subPath: wg0.conf
          - name: host-volumes
            mountPath: /lib/modules
          - mountPath: /etc/sysctl.conf
            name: sysctl
            subPath: sysctl.conf
        ports:
        - containerPort: 51820
          protocol: UDP
        resources:
          requests:
            memory: 10Mi
            # cpu: "100m"
          limits:
            memory: "10Mi"
            # cpu: "200m"

      # this is for coredns
      - args:
        {{- if $devInfo}}
        - --dev-info
        - {{ $devInfo }}
        {{- end}}
        - --addr
        - 0.0.0.0:17171
        {{- if $tlsCertSecName}}
        - --tls-addr
        - 0.0.0.0:17172
        {{- end}}
        - --corefile
        - /etc/coredns/Corefile
        - --debug
        image: ghcr.io/kloudlite/operator/components/coredns:v1.0.5-nightly
        imagePullPolicy: IfNotPresent
        name: coredns
        resources:
          limits:
            # cpu: 100m
            memory: 20Mi
          requests:
            # cpu: 100m
            memory: 20Mi
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_BIND_SERVICE
            {{- /* drop: */}}
            {{- /* - all */}}
          {{- /* readOnlyRootFilesystem: true */}}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File

        {{- if $tlsCertSecName}}
        env:
        - name: TLS_CERT_FILE_PATH
          value: /etc/vpn-device/tls
        {{- end}}

        volumeMounts:
        - mountPath: /etc/coredns
          name: config-volume
          readOnly: true
        - mountPath: /etc/coredns/custom
          name: custom-config-volume
          readOnly: true
        {{- if $tlsCertSecName}}
        - mountPath: /etc/vpn-device/tls
          name: tls-certs
          readOnly: true
        {{- end}}

      # end of coredns
      volumes:
        - name: sysctl
          secret:
            items:
            - key: sysctl
              path: sysctl.conf
            secretName: "wg-configs-{{$name}}"
        - name: wg-config
          secret:
            secretName: "wg-configs-{{$name}}"
            items:
              - key: server-config
                path: wg0.conf
        - name: host-volumes
          hostPath:
            path: /lib/modules
            type: Directory
        {{- if $tlsCertSecName}}
        - name: tls-certs
          secret:
            secretName: {{ $tlsCertSecName }}
        {{- end}}


        # for coredns
        - configMap:
            defaultMode: 420
            items:
            - key: Corefile
              path: Corefile
            name: "wg-dns-{{$name}}"
          name: config-volume
        - configMap:
            defaultMode: 420
            name: coredns-custom
            optional: true
          name: custom-config-volume

      # for coredns
      dnsPolicy: Default
      priorityClassName: system-cluster-critical
      restartPolicy: Always
      schedulerName: default-scheduler
      terminationGracePeriodSeconds: 30

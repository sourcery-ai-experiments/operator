{{- /*gotype: github.com/kloudlite/operator/operators/clusters/internal/templates.ClusterJobVars*/ -}}
{{ with . }}
apiVersion: crds.kloudlite.io/v1
kind: Job
metadata:
  name: {{.JobMetadata.Name}}
  namespace: {{.JobMetadata.Namespace}}
  labels: {{.JobMetadata.Labels | default dict | toYAML | nindent 4}}
  annotations: {{.JobMetadata.Annotations | default dict | toYAML | nindent 4}}
  ownerReferences: {{.JobMetadata.OwnerReferences | default list | toYAML| nindent 4}}
spec:
  onApply:
    backOffLimit: 1
    podSpec:
      restartPolicy: Never

      nodeSelector: {{ .NodeSelector | default dict | toYAML | nindent 8 }}
      tolerations: {{ .Tolerations | default list | toYAML | nindent 8 }}
      containers:
        - name: iac
          image: {{.JobImage}}
          imagePullPolicy: Always

          resources:
            requests:
              cpu: 500m
              memory: 1000Mi
            limits:
              cpu: 500m
              memory: 1000Mi

          env:
            - name: KUBE_IN_CLUSTER_CONFIG
              value: "true"

            - name: KUBE_NAMESPACE
              value: {{.TFStateSecretNamespace}}

          command:
            - bash
            - -c
            - |+
              set -o pipefail
              set -o errexit

              eval $DECOMPRESS_CMD

              pushd "$TEMPLATES_DIR/kl-target-cluster-aws-only-masters"
              envsubst < state-backend.tf.tpl > state-backend.tf

              terraform init -reconfigure -no-color 2>&1 | tee /dev/termination-log
              terraform workspace select --or-create {{.TFStateSecretName}}

              cat > values.json <<EOF
              {{.ValuesJSON}}
              EOF

              terraform plan -parallelism=2 -out tfplan --var-file ./values.json -no-color 2>&1 | tee /dev/termination-log
              terraform apply -parallelism=2 -no-color tfplan 2>&1 | tee /dev/termination-log

              terraform state pull | jq '.outputs' -r > outputs.json

              cat outputs.json

              kubectl apply -f - <<EOF
              apiVersion: v1
              kind: Secret
              metadata:
                name: {{.ClusterSecretName}}
                namespace: {{.ClusterSecretNamespace}}
              data:
                kubeconfig: $(cat outputs.json | jq '.kubeconfig.value')
                k3s_params: $(cat outputs.json | jq -r '."kloudlite-k3s-params".value' | base64 | tr -d '\n')
                k3s_agent_token: $(cat outputs.json | jq -r '.k3s_agent_token.value' | base64 | tr -d '\n')
              EOF
              exit 0
  onDelete:
    backOffLimit: 1
    podSpec:
      restartPolicy: Never
      containers:
        - name: iac
          image: {{.JobImage}}
          imagePullPolicy: "Always"

          resources:
            requests:
              cpu: 500m
              memory: 1000Mi
            limits:
              cpu: 500m
              memory: 1000Mi

          env:
            - name: KUBE_IN_CLUSTER_CONFIG
              value: "true"

            - name: KUBE_NAMESPACE
              value: {{.TFWorkspaceSecretName | squote}}

          command:
            - bash
            - -c
            - |+
              set -o pipefail
              set -o errexit

              eval $DECOMPRESS_CMD

              {{- /* pushd "$TEMPLATES_DIR/{{.CloudProvider}}/vpc" */}}
              pushd "$TEMPLATES_DIR/gcp/vpc"

              envsubst < state-backend.tf.tpl > state-backend.tf

              terraform init -reconfigure -no-color 2>&1 | tee /dev/termination-log
              terraform workspace select --or-create {{.TFWorkspaceName}}

              cat > values.json <<EOF
              {{.ValuesJSON}}
              EOF

              terraform plan -parallelism=2 --destroy --var-file ./values.json -out=tfplan -no-color 2>&1 | tee /dev/termination-log
              terraform apply -parallelism=2 -no-color tfplan 2>&1 | tee /dev/termination-log
              kubectl delete secret/{{.ClusterSecretName}} -n {{.ClusterSecretNamespace}} --ignore-not-found=true
              exit 0
{{ end }}

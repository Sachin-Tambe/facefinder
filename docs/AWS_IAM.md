# AWS IAM (least privilege, §36) — attach to the API/worker role, never root keys
```json
{ "Version": "2012-10-17", "Statement": [{ "Effect": "Allow",
  "Action": ["rekognition:CreateCollection","rekognition:DeleteCollection","rekognition:IndexFaces","rekognition:SearchFacesByImage","rekognition:DetectFaces","rekognition:DeleteFaces","rekognition:ListFaces","rekognition:DescribeCollection"],
  "Resource": "*" }] }
```
Rekognition collections are not resource-ARN scoped in most regions; compensate with the `evt_` prefix convention + separate AWS account/role for this workload. R2 keys: scope to the single private bucket, server-side only.

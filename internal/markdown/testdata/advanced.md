# Advanced Document

> This is a quoted paragraph.
> It spans two lines.

| name | role |
|---|---|
| ink | previewer |

<div class="note">
raw html is preserved
</div>

```mermaid
flowchart LR
  APIGW["API Gateway"] --> LambdaA["Lambda"]
  LambdaA --> SQS["SQS"]
  SQS --> LambdaB["Lambda"]
  LambdaB --> RDS[("RDS")]
```

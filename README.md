# hfc: Helper for CloudFormation

hfc is a strongly opinionated tool to deploy Go applications that meet very
specific requirements with AWS CloudFormation. It is made public for the sake
of other personal projects that rely on this specific workflow, as an
improvement over copy-pasting a shell script between multiple projects.

I do not recommend hfc for general use. Try [AWS SAM][sam] instead, and see the
[go-al2][go-al2] example in particular.

[sam]: https://aws.amazon.com/serverless/sam/
[go-al2]: https://github.com/aws-samples/sessions-with-aws-sam/tree/master/go-al2

# ktf

ktf is a tool to convert kubernetes yaml files to terraform.

The simplest way to do this would be to use the terraform kubernetes provider's
generic `kubernetes_manifest` type. A better approach would be to actually read
the yaml and use the actual corresponding resource, only falling back to
`kubernetes_manifest` for things like CRDs that aren't representable in a better
way. The downside of that is almost every type of resource would need some
custom logic.

The goal for this tool is to sit somewhere in the middle, and try and
idiomatically convert a subset of common resources while falling back to
`kubernetes_manifest` for anything else.

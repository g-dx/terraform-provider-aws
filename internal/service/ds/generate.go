//go:generate go run ../../generate/listpages/main.go -ListOps=DescribeDirectories,DescribeRegions
//go:generate go run ../../generate/tags/main.go -ListTags -ListTagsInIDElem=ResourceId -ServiceTagsSlice -TagOp=AddTagsToResource -TagInIDElem=ResourceId -UntagOp=RemoveTagsFromResource -UpdateTags -CreateTags
//go:generate go run ../../generate/servicepackage/main.go
// ONLY generate directives and package declaration! Do not add anything else to this file.

package ds

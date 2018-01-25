# Grouping functions based on labels 

The motivation for this comes from the need to list all functions using the same package. Also, from the need to list all functions using the same environment.
Today the function objects have references to both these objects. 
So, the obvious straight forward way to get a list of all functions using a package, for example, is to get a list of all functions first, iterate through each of them extract the package names and append the function names to a list whose packageRef are common.
But, this approach may not scale very well.

Enter Labels. Kubernetes documentation claims that it will eventually index and reverse-index labels for efficient queries and watches. 
The documentation also says that "label selector" is the core grouping primitive in Kubernetes.

To satisfy the above use case of listing all functions using the same package and listing all functions using the same environment, this solution propose to introduce 2 labels into function object - one for package, other for env.
 
## Details
 
 TODO
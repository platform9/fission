# Grouping functions based on labels 

## Summary
The motivation for this comes from the need to list all functions using the same package. Also, from the need to list all functions using the same environment.
Today, the function objects have references to both these objects in their `Spec` field. 
The obvious straight forward way to get a list of all functions using a package, for example, is to get a list of all functions first, iterate through each of them, extract the package names and append the function names to a list whose packageRef are the same, finally return this list.
But, this approach may not scale very well when there are way too many functions in the cluster, out of which, only a small fraction of them share a package.

Kubernetes documentation claims that it will eventually index and reverse-index labels for efficient queries and watches. 
The documentation also says that "label selector" is the core grouping primitive in Kubernetes.

To satisfy the above stated use cases, this solution proposes to introduce 2 labels into the function object - one for package, other for env.
Accordingly, instead of fission code getting all functions and then filtering out the ones that dont share a package, we let kubernetes pick only the functions that have the same package label.

## Note
1.  I think this document only serves as a good summary before starting to review this change. I don't think we should retain this as part of the final codebase.
    
## Implementation details of using labels
1.  Everytime a user creates a function, the fission cli code adds 2 labels to the function CRD, one for package and the other for environment. 
    ```go
    labels := make(map[string]string)
    labels["package"] = packageName
    labels["environment"] = envName
    ```
    
2.  Everytime a user updates a function to refer to a different package or a different environment, correspondingly, the fission cli modifies the labels on the function object.

3.  If a user wants to view a list of orphan packages (not referenced by any function), he can do so with the following cli
    ```bash
      $ fission package list --orphan
    ```
    The fission cli gets a list of all packages and then gets the functions used by each package using the label selector. Finally outputs all the orphan packages.
    
4.  If a user wants to delete all orphan packages, he can do so with the following cli
    ```bash
      $ fission package delete --orphan
    ```
    The fission cli executes the same code as above, in addition, issuing a delete request to each of those orphan packages.
    
 ## Backward compatibility
1.  Functions created with older releases of fission will not have these labels. So, in a scenario where two functions created with and without labels share a package, then, 
    getting all functions using the same package with label selectors will give wrong results.
    To get around this issue, we must let the users know to first perform an update on all existing functions with the following cli (this just updates function objects with labels)
    and then do a helm upgrade to the new release. (// TBD : I do not know if this is a good idea, but couldn't think of a better approach.)
    ```bash
    $ fission function update --labels
    ```

2.  Even if the user forget to update labels before an upgrade, if the user executes either of `fission package list --orphan` or `fission package delete --orphan`,
    fission cli takes that as an opportunity to label all functions before listing or deleting orphan packages. This is needed because the function to list all orphans 
    and delete them relies on grabbing functions using label selectors. 
    

  

 
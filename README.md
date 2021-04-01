    # Terraformer-ator - beautify your terrafomer code

    ## What does this solve? 
    
    You have a resources in your infrastructure that was not created from Terraform originally. This can be for various reasons. Do you have dreams of taking the `Terraformer` output and turn it into a _beautified_ format with all dependent resources all in one nicely wrapped tf file per resource? Maybe `Terraformer-ator` can help make your dreams become true.  

    ## How does `terraformer-ator` beautify terraformer?
    
    Given a terraform resource type, i.e. `google_compute_resource` or i.e. `google_compute_target_https_proxy`, this will create a terraform file with the hcl syntax and any of it's dependencies into one file named after the resource configuration block name. 

    For example, if I have a "google_compute_resource" instance, pulled from Terraformer with the block below, given the syntax below pulled from Terraformer: 


    ```
    resource "aws_instance" "blue_city_rising" {
      ami                    = "ami-408c7f28"
      instance_type          = "t1.micro"
      monitoring             = true
      vpc_security_group_ids = [
          "tg-14abcf",
      
      tags          = {
        Name        = "Application Server"
        Environment = "production"
      }
       root_block_device {
        delete_on_termination = false
      }
    }
    ```

    ...this will take the following syntax from above and put the terraform block into the `blue_city_rising.tf` file with any other dependent resource blocks that use `aws_instance.blue_city_rising` object, nicely packed into one file (i.e. the `blue_city_rising.tf` file).

    # What you'll need:
     
    • Golang version 1.16 or above installed. 

    • A git clone of, [Terraformer](https://github.com/GoogleCloudPlatform/terraformer/). Later in this readme, we will: 
    1. Make a modificaton to the Terraformer code and rebuild the source. This will remove "tfer--" and keep the "-"'s the same instead of decoding "-"s to a different format. 
    2. Invoke Terraformer to import _all_ resources to ensure any dependency Terraformer supports is not missed. 
    <p>See the "Terraformer setup" section below. 

    • A `.tf` file that has all your tls certificates in efforts to reference the certs appropriately. For example, create a certificate.tf file with the data blocks referencing the tls certificates below: 

    ```
    data "google_compute_ssl_certificate" "asos_com_tls_cert" {
      name = "asos-com-expires-june-01-2024"
    }

    data "google_compute_ssl_certificate" "pre_prod_asos_com_tls_cert" {
      name = "pre-prod-asos-com-expires-june-09-2024"
    }
    ```

    • A directory created to point all the newly created .tf files to. 

    # Terraformer setup
    1. Perform a clone of [Terraformer](https://github.com/GoogleCloudPlatform/terraformer/). 
    ```
    git clone git@github.com:GoogleCloudPlatform/terraformer.git
    ```
    2. Open up the `terraformutils/hcl.go` file. Find the `var unsafeChars` line and replace....
    <p><b><u>From:</b></u>

    ```
    var unsafeChars = regexp.MustCompile(`[^0-9A-Za-z_]`)
    ```

    <b><u>To:</u></b>
    ```
    var unsafeChars = regexp.MustCompile(`[^0-9A-Za-z_-]`)
    ```

    then remove the line containing:

    ```
    name = “tfer—“ + name
    ```

    3. Save the file and navigate to the root of the directory. Run the following: 
    ```
    go build 
    ```

    4. Create a directory, and run Terraformer with the given target directory that was just created. We also create a `tf-beautifier-dir/` directory where the nicely formatted terraform files will be generated. 

    ```
    $ mkdir tf-beautifier-dir/
    $ mkdir terraformer-output-dir/; cd terraformer-output-dir/
    $ /path/to/your/built/terraformer import google --regions=global,<your_regions> --projects=<your_project> --resources="*"

    ```
    # Running Terraformer-ator 

    1. Naviate to where Terraformer-ator is and build the source: 
    ```
    $ cd terraformer-ater/
    $ go build
    $ cd ..
    $ cd wrapper/
    $ go build 
    ```

    After running `go build` in both the `terraformer-ater/` and `wrapper/` directories,  you should see a binary file created in each directory that we will run. 

    2. Run the following command to invoke the `terraformer-ater/` wrapper. 

    ```
    $ wrapper/wrapper terraformer-output-dir/ "google_compute_instance" tf-beautifier-dir/  tls-certificates.tf
    ```

    Laying out what the above syntax shows exactly: 

    ```
    $ wrapper/wrapper terraformer-output-dir/ "google_compute_instance" tf-beautifier-dir/  tls-certificates.tf
    ```
    > * `wrapper/wrapper` - invokes terrafor-ator via the wrapper binary
    > * `terraformer-output-dir/` - the directory terraformer was ran in
    > * `"google_compute_instance"` - the target resource type we want, for each resource type, a new file will be created that will include any dependent resources, all in one file. 
    > * `tf-beautifier-dir/` - the target directory our beautified .tf files will be generated in
    > *  `tls-certificates.tf` - the tf file where the tls certificates.tf live. For more details, please refer to example earlier in this readme. 

    3. After invoking the command in the previous step, and navigating to  the `tf-beautifier-dir/` directory, you will now see the tf files, one  file per "google_compute_instance" and any other dependent resources that use that resource all in one file. 

    We are looking for contributors, add your PR today! 
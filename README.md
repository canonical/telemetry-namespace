# Telemetry Template Repository

This repository is a template to be used to create repostries for Go-based projects in the telemetry team. It follows the Golang [project structure](#project-structure) as well as including the SSDLC workflows needed to generate artefacts. We also add the required SECURITY.md file for SSDLC and a template `.golangci.yml` file for linting.  

## SSDLC Workflows
This repository also contains the workflows for the SSDLC artefacts required by each team. They include:
* TICS Static Analysis: the project must be registered first and the `TICSAUTHTOKEN` must be added as a secret. (Check the [docs](https://canonical-telemetry-internal.readthedocs-hosted.com/en/latest/Development/CICD/intro/) for more details)
* Vulnerability Scan: we currently use Trivy for our analysis, but are waiting on the sbomber tool to be Github-compatible. The results ate then added to the home directory where they can be stored for later.
* SBOM Generation: we again use Trivy for it, but will also be replaced by sbomber.


The workflows will `on` keyword will be commented out as they will fail without any actual code. They can be activated by uncommenting that section from the YAML files. We also include a template SECURITY.md file that must be added if the repository is public and must address the security status of the project as well as the reporting mechanism for vulnerabilities.

## Project Structure

In general, projects should follow [this](https://github.com/golang-standards/project-layout) project layout:  
* *`/cmd:`* The main applications for the project.  
* *`/internal:`* Private application and library code. This is the code you don't want others importing in their applications or libraries.  
* *`/pkg:`* Library code that's ok to use by external applications.  
* *`/vendor:`* Application dependencies managed with *`go mod vendor`*.  
* *`/api:`* OpenAPI/Swagger specs, JSON schema files, protocol definition files.  
* *`/web:`* Web application specific components: static web assets, server side templates and SPAs.  
* *`/scripts:`* Scripts to perform various build, install, analysis, etc operations which helps keep the root level *`Makefile`* small and simple.  
* *`/snap:`* If the project is snapped.  
* *`/charm:`* If the project is charmed.  
* *`/docs:`* Design and user documents (in addition to your *`godoc`* generated documentation).

### Packages

* The code in *`/internal`* and *`/pkg`* should be structured in packages (see [Package oriented design](https://github.com/ardanlabs/gotraining/blob/master/topics/go/design/packaging/README.md)).  
* Packages should have a clear responsibility, and present an exported interface with a relatively consistent level of abstraction.  
* Packages must be named with the intent to describe what they provide.

## Linting
Golang projects must undergo linting before being pushed we use [golangci](https://golangci-lint.run/) to customize our linting and add or remove linters based on the use case. We provide a template `.golangci.yml` file to use as a reference.
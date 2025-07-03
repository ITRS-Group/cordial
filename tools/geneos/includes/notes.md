geneos init [PROFILE]
geneos add gateway --profile PROFILE
geneos deploy --profile PROFILE

geneos set gateway abc --profile PROFILE



Include file type:

* Functional
  * Read Only
  * Upgradeable
  * Managed

* Operational
  * "Glue"
  * Managed

* Configuration
  * Contains defaults
  * Once created is user property
  * Unmanaged

* Template
  * Examples / Frameworks
  * Copy to edit
  * Upgradeable
  * Managed

Permission Setup/Section controlled
Authentication defaults

* Admin account, auto generated password
* All other users, geneos account

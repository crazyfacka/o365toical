# o365toical

Grabs your Office 365 calendar and provides an iCal endpoint to it.

Uses the Microsoft Graph API to live grab the following calendar week and return an iCal formatted output. The following 3 weeks are cached daily as to provide a quick API response. On my tests, the Microsoft Graph API is taking an average of 2-3 seconds per calendar week.

## Requirements

* Register an App within your [Azure Portal](https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationsListBlade)
  * Must have the `Calendars.Read`, `User.Read` and `offline_access` permissions
  * You should be able to extract `client_id`, `secret` and `tenant`
  * Don't forget to add a valid **Redirect URL**
* Docker
* A folder on where to store attachments
* A MariaDB/MySQL installation (tested with MariaDB 10.5)

## Build & configure

### Build and copy sample files

```
$ docker build -t account/o365tocal .
$ mv sample_config.json config.json
```

### Edit configuration

**client_id:** Client ID retrieved from the Azure Portal<br/>
**secret:** Secret retrieved from the Azure Portal<br/>
**tenant:** Tenant retrieved from the Azure Portal<br/>
**redirect_url:** The URL to where to redirect after successful authentication<br/>
**attachments_dir:** Directory on where to store the attachments<br/>
**mysql**<br/>
&nbsp;&nbsp;&nbsp;&nbsp;*user:* User with which to connect to the DB<br/>
&nbsp;&nbsp;&nbsp;&nbsp;*password:* Password corresponding to the user<br/>
&nbsp;&nbsp;&nbsp;&nbsp;*host:* Host of the database<br/>
&nbsp;&nbsp;&nbsp;&nbsp;*schema:* Schema on where to store all the information

## Run

```
$ docker run --name o365 -p 5000:5000 --restart unless-stopped -v /confs/o365/config.json:/app/config.json -v /data/o365:/files crazyfacka/o365toical

O365 to iCal build from 2021-06-23_2206

   ____    __
  / __/___/ /  ___
 / _// __/ _ \/ _ \
/___/\__/_//_/\___/ v4.2.2
High performance, minimalist Go web framework
https://echo.labstack.com
____________________________________O/_______
                                    O\
⇨ http server started on [::]:5000
```

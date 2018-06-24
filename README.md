## webanalyse

Analyse a webpage.

## Application URL

http://localhost:8080

## How to run app directly
There is an executable in the repository.Clone the repo and run the executable.


    ./webanalyse

Access the [url](http://localhost:8080).   


## How to run app from source
1. Clone this app and ``` cd ``` into it.
2. Ensure dependencies ``` dep init and dep ensure  ```
3. Build :
    * ``` go build . ```
    * After this step, executable webanalyse is generated in same directory
4. Run :
    * ``` ./webanalyse ```    
5. Now access the [app](http://localhost:8080).

## Few points
1. Used golang, as I wanted to get a quick working app.
2. The UI is a basic one without optimizations.
3. We are making HEAD request to check if a website is accessible.
4. Links for a subdomain on a domain are considered external to the website.
5. External links which dont respond within a certain time are considered inaccessible.
6. Anchor links with href as a javascript func are also considered internal links.
7. Used the rakyl statik package to bundle assets into the executable.
8. Used goquery for document parsing.
9. Showing errors on a separate error page.

## Further Improvements
1. HTTP static asset serving and cache control headers.
2. Need to check for a better worker pool package and work splitting to improve response time.Its slow now.
3. Write tests.
4. Use other url parsing package such as [urlx](https://github.com/goware/urlx) for accepting url's without scheme.
5. Use a spinner for showing some progress to user.

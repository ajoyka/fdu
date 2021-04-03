# fdu 

Fast utility for running classic linux/MacOS disk usage utility concurrently 
Written in golang

Can be used for finding duplicate image files. Useful for collecting and organizing pictures from various folders. 

Generates json files with metadata to help consolidate and identify locations of all pictures

Steps:

go get github.com/ajoyka/fdu 
go build 
./fdu <dirs to traverse>... 

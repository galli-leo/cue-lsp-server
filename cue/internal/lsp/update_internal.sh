#!/bin/sh

for dir in `ls -d internal/vendored/*/`;                                  
do                                                                        
    echo "Updating:";                                                     
    NAME=`basename $dir`;                                                
    echo "Name: " $NAME;                                                 
    REPO=`cat internal/vendored/$NAME.repo`;                             
    echo "Repo: " $REPO;                                                 
    FCMD=`cat internal/vendored/$NAME.cmd`;                              
    echo "File cmd: " $FCMD;                                             
    DIRS=`cat internal/vendored/$NAME.dirs`;                             
    echo "Directories: " $DIRS;                                          
    VERSION=`cat internal/vendored/$NAME.version`;                       
    echo "Version: " $VERSION;                                           
    echo "Cleaning up";                                                   
    rm -rf $dir*;                                                        
    TMPDIR=`mktemp -d`;                                                   
    echo "Temp dir: " $TMPDIR;                                           
    git clone $REPO $TMPDIR;                                            
    git --git-dir=$TMPDIR/.git --work-tree=$TMPDIR checkout $VERSION;  
    echo "Copying Files";                                                 
    for subdir in $DIRS;                                                 
    do                                                                    
        echo mkdir -p `dirname $dir$subdir`;                            
        mkdir -p `dirname $dir$subdir`;                                 
        cp -r $TMPDIR/internal/$subdir $dir$subdir;                   
    done;                                                                 
    for file in `find $dir -type f`;                                     
    do                                                                    
        CMD=`echo $FCMD $file`;                                         
        echo $CMD;                                                       
        bash -c "$CMD";                                                  
    done;                                                                 
    rm -rf $TMPDIR;                                                      
    make fmt;                                                             
done
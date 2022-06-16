{\rtf1\ansi\ansicpg1252\cocoartf2638
\cocoatextscaling0\cocoaplatform0{\fonttbl\f0\fnil\fcharset0 Menlo-Bold;\f1\fnil\fcharset0 Menlo-Regular;}
{\colortbl;\red255\green255\blue255;\red86\green219\blue233;\red34\green79\blue188;\red97\green0\blue1;
\red88\green229\blue64;\red20\green153\blue2;\red204\green203\blue60;\red103\green0\blue109;}
{\*\expandedcolortbl;;\cssrgb\c38881\c88035\c93072;\csgenericrgb\c13206\c30848\c73913;\cssrgb\c46056\c0\c0;
\cssrgb\c39489\c89858\c31572;\cssrgb\c0\c65000\c0;\cssrgb\c83612\c82453\c29972;\cssrgb\c48749\c0\c50223;}
\margl1440\margr1440\vieww11520\viewh8400\viewkind0
\pard\tx560\tx1120\tx1680\tx2240\tx2800\tx3360\tx3920\tx4480\tx5040\tx5600\tx6160\tx6720\pardirnatural\partightenfactor0

\f0\b\fs6 \cf2 \cb3 \CocoaLigature0 package
\f1\b0 \cf1  main\
\

\f0\b \cf2 import
\f1\b0 \cf1  (\
        \cf4 "fmt"\cf1 \
\
        \cf4 "github.com/360EntSecGroup-Skylar/excelize"\cf1 \
)\
\
\cf5 func\cf1  main() \{\
\cb6         \cb3 \
        fmt.Print(\cf4 "------------------------------------------------------------- \\n "\cf1 )\
        fmt.Print(\cf4 "------------------------------ MAIN CHAIN -------------------- \\n "\cf1 )\
        fmt.Print(\cf4 "------------------------------------------------------------- \\n "\cf1 )\
\
        xlsx, err := excelize.OpenFile(\cf4 "/root/remote/mainchainbc.xlsx"\cf1 )\
        
\f0\b \cf7 if
\f1\b0 \cf1  err != \cf4 nil\cf1  \{\
                fmt.Println(err)\
                \cf8 return\cf1 \
        \}\
        fmt.Print(\cf4 "----------------------  MarketMatching ------------------- \\n "\cf1 )\
        rows := xlsx.GetRows(\cf4 "MarketMatching"\cf1 )\
        
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , row := 
\f0\b \cf7 range
\f1\b0 \cf1  rows \{\
                
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , colCell := 
\f0\b \cf7 range
\f1\b0 \cf1  row \{\
                        fmt.Print(colCell, \cf4 "\\t"\cf1 )\
                \}\
                fmt.Println()\
        \}\
\
        fmt.Print(\cf4 "----------------------  FirstQueue ------------------- \\n "\cf1 )\
        rows = xlsx.GetRows(\cf4 "FirstQueue"\cf1 )\
        
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , row := 
\f0\b \cf7 range
\f1\b0 \cf1  rows \{\
                
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , colCell := 
\f0\b \cf7 range
\f1\b0 \cf1  row \{\
                        fmt.Print(colCell, \cf4 "\\t"\cf1 )\
                \}\
                fmt.Println()\
        \}\
\
        fmt.Print(\cf4 "----------------------  SecondQueue ------------------- \\n "\cf1 )\
        rows = xlsx.GetRows(\cf4 "SecondQueue"\cf1 )\
        
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , row := 
\f0\b \cf7 range
\f1\b0 \cf1  rows \{\
                
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , colCell := 
\f0\b \cf7 range
\f1\b0 \cf1  row \{\
                        fmt.Print(colCell, \cf4 "\\t"\cf1 )\
                \}\
                fmt.Println()\
        \}\
\
        fmt.Print(\cf4 "----------------------  PowerTable ------------------- \\n "\cf1 )\
        rows = xlsx.GetRows(\cf4 "PowerTable"\cf1 )\
        
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , row := 
\f0\b \cf7 range
\f1\b0 \cf1  rows \{\
                
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , colCell := 
\f0\b \cf7 range
\f1\b0 \cf1  row \{\
                        fmt.Print(colCell, \cf4 "\\t"\cf1 )\
                \}\
                fmt.Println()\
        \}\
\
        fmt.Print(\cf4 "----------------------  RoundTable ------------------- \\n "\cf1 )\
        rows = xlsx.GetRows(\cf4 "RoundTable"\cf1 )\
        
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , row := 
\f0\b \cf7 range
\f1\b0 \cf1  rows \{\
                
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , colCell := 
\f0\b \cf7 range
\f1\b0 \cf1  row \{\
                        fmt.Print(colCell, \cf4 "\\t"\cf1 )\
                \}\
                fmt.Println()\
        \}\
\
        fmt.Print(\cf4 "----------------------  OverallEvaluation ------------------- \\n "\cf1 )\
        rows = xlsx.GetRows(\cf4 "OverallEvaluation"\cf1 )\
        
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , row := 
\f0\b \cf7 range
\f1\b0 \cf1  rows \{\
                
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , colCell := 
\f0\b \cf7 range
\f1\b0 \cf1  row \{\
                        fmt.Print(colCell, \cf4 "\\t"\cf1 )\
                \}\
                fmt.Println()\
        \}\
\
        fmt.Print(\cf4 "------------------------------------------------------------- \\n "\cf1 )\
        fmt.Print(\cf4 "------------------------------ SIDE CHAIN -------------------- \\n "\cf1 )\
        fmt.Print(\cf4 "------------------------------------------------------------- \\n "\cf1 )\
\cb6         \cb3 \
        xlsx, err = excelize.OpenFile(\cf4 "/root/remote/sidechainbc.xlsx"\cf1 )\
        
\f0\b \cf7 if
\f1\b0 \cf1  err != \cf4 nil\cf1  \{\
                fmt.Println(err)\
                \cf8 return\cf1 \
        \}\
        fmt.Print(\cf4 "----------------------  MarketMatching ------------------- \\n "\cf1 )\
        rows = xlsx.GetRows(\cf4 "MarketMatching"\cf1 )\
        
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , row := 
\f0\b \cf7 range
\f1\b0 \cf1  rows \{\
                
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , colCell := 
\f0\b \cf7 range
\f1\b0 \cf1  row \{\
                        fmt.Print(colCell, \cf4 "\\t"\cf1 )\
                \}\
                fmt.Println()\
        \}\
\
        fmt.Print(\cf4 "----------------------  FirstQueue ------------------- \\n "\cf1 )\
        rows = xlsx.GetRows(\cf4 "FirstQueue"\cf1 )\
        
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , row := 
\f0\b \cf7 range
\f1\b0 \cf1  rows \{\
                
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , colCell := 
\f0\b \cf7 range
\f1\b0 \cf1  row \{\
                        fmt.Print(colCell, \cf4 "\\t"\cf1 )\
                \}\
                fmt.Println()\
        \}\
\
        fmt.Print(\cf4 "----------------------  RoundTable ------------------- \\n "\cf1 )\
        rows = xlsx.GetRows(\cf4 "RoundTable"\cf1 )\
        
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , row := 
\f0\b \cf7 range
\f1\b0 \cf1  rows \{\
                
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , colCell := 
\f0\b \cf7 range
\f1\b0 \cf1  row \{\
                        fmt.Print(colCell, \cf4 "\\t"\cf1 )\
                \}\
                fmt.Println()\
        \}\
\
        fmt.Print(\cf4 "----------------------  OverallEvaluation ------------------- \\n "\cf1 )\
        rows = xlsx.GetRows(\cf4 "OverallEvaluation"\cf1 )\
        
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , row := 
\f0\b \cf7 range
\f1\b0 \cf1  rows \{\
                
\f0\b \cf7 for
\f1\b0 \cf1  \cf4 _\cf1 , colCell := 
\f0\b \cf7 range
\f1\b0 \cf1  row \{\
                        fmt.Print(colCell, \cf4 "\\t"\cf1 )\
                \}\
                fmt.Println()\
        \}\
\
\}\
\
}
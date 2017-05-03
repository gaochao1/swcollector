#!/bin/bash


###判断是os6还是os7
function judge_os()
{
	fgrep 'CentOS release 6' /etc/issue -q && return 6
	fgrep 'CentOS Linux release 7'  /etc/redhat-release -q && return 7 
}

###将ip地址由10进制换为16进制
function chan()
{
        num1=`echo "$1" |awk -F. '{print $1}'`
        num2=`echo "$1" |awk -F. '{print $2}'`
        num3=`echo "$1" |awk -F. '{print $3}'`
        num4=`echo "$1" |awk -F. '{print $4}'`
        printf "%x.%x.%x.%x" "$num1" "$num2" "$num3" "$num4"
}


###新建iptables的链
function new_chain()
{
	iptables -L "$1" &> /dev/null
	if [ "$?" -ne 0 ];then
		iptables -N "$1"
	fi
}


###为新建的链插入规则
function insert_input_rule()
{	
	chain_name="$1"
	device_name="$2"	
	rule_num=`echo $(iptables -L "$chain_name" -nv | egrep "$device_name" | fgrep Chain -v | fgrep pkts -v| wc -l)`
	if [ "$rule_num" -eq 0 ];then
	    	iptables -I INPUT -i "$device_name" -j "$chain_name"
	fi
}

###为新建的链插入规则
function insert_output_rule()
{       
        chain_name="$1"
        device_name="$2"        
        rule_num=`echo $(iptables -L "$chain_name" -nv | egrep "$device_name" | fgrep Chain -v | fgrep pkts -v| wc -l)`
        if [ "$rule_num" -eq 0 ];then
                iptables -I OUTPUT -o "$device_name" -j "$chain_name"
        fi
}




###

function chain_bunding_in()
{
	chain_name="$1"
	device_name="$2"
	iptables -L "$chain_name" -nv | egrep "$device_name" -q || (iptables -A "$chain_name" -i "$device_name")
}


##

function chain_bunding_out()
{
        chain_name="$1"
        device_name="$2"
	iptables -L "$chain_name" -nv | egrep "$device_name" -q || (iptables -A "$chain_name" -o "$device_name")	
}

##

function chain_bunding_in_net()
{
	chain_name="$1"
	device_name="$2"
	net_name="$3"
	iptables -L "$chain_name" -nv | egrep "$device_name" -q || (iptables -A "$chain_name" -i "$device_name" -s "$net_name")
}

##
function chain_bunding_out_net()
{
    chain_name="$1"
    device_name="$2"
	net_name="$3"
	iptables -L "$chain_name" -nv | egrep "$device_name" -q || (iptables -A "$chain_name" -o "$device_name" -d "$net_name")
}

###判断是否有ipv6的地址
function judge_has_ipv6()
{
	ip addr show | fgrep inet6 | fgrep global -q && return 0 || return 1
}

###ipv6的流量采集函数
function collect_ipv6()
{
	flag=`echo $(ip addr show | fgrep inet6 | fgrep global  | cut -d':' -f1| sed 's/inet6//')`
	ipv6_netcard=`echo $(cat /proc/net/if_inet6 | fgrep -e "$flag" | awk '{print $NF}')`
	ip6tables -N in_ipv6
	ip6tables -N out_ipv6
	ip6tables -I INPUT -i "$ipv6_netcard" -j in_ipv6
	ip6tables -I OUTPUT -o "$ipv6_netcard" -j out_ipv6
	ip6tables -A in_ipv6 -i "$ipv6_netcard"
	ip6tables -A out_ipv6 -o "$ipv6_netcard"	
}


###ipv4的流量采集
function collect_ipv4()
{
	ip_all=(`ip addr show | grep "inet\>"|grep "127.0.0.1" -v | grep "secondary" -v | grep "brd"| grep " 172.1" -v| awk '{print $2}'| awk -F/ '{print $1}'`)
	dev_all=(`ip addr show | grep "inet\>"|grep "127.0.0.1" -v | grep "secondary" -v | grep "brd"| grep " 172.1" -v | awk '{print $NF}'`)
	if ! [ "$ip_all" = ""  ] ; then
	    for ((i=0; i<${#ip_all[@]}; ++i))
		do
		    net=`echo "${ip_all[$i]}" |sed -r 's@[0-9]+$@0/24@'`
		    ip_hex=`chan "${ip_all[$i]}"`
		    if ! [ "$net" = ""  ] ; then
			new_chain traffic_in_${ip_hex}
			new_chain traffic_out_${ip_hex}
			new_chain traffic_lan_in_${ip_hex}
			new_chain traffic_lan_out_${ip_hex}
		
			insert_input_rule  traffic_in_${ip_hex}  "${dev_all[$i]}" 
			insert_output_rule  traffic_out_${ip_hex}  "${dev_all[$i]}" 
			insert_input_rule  traffic_lan_in_${ip_hex}  "${dev_all[$i]}" 
			insert_output_rule  traffic_lan_out_${ip_hex}  "${dev_all[$i]}" 
		
			chain_bunding_in traffic_in_${ip_hex}  "${dev_all[$i]}" 
			chain_bunding_out traffic_out_${ip_hex}  "${dev_all[$i]}" 
			chain_bunding_in_net traffic_lan_in_${ip_hex}  "${dev_all[$i]}" "$net"
			chain_bunding_out_net traffic_lan_out_${ip_hex}  "${dev_all[$i]}" "$net"
		    fi
		done
	fi
}


#######流量采集###################################
collect_ipv4
judge_has_ipv6
if [ "$?" -eq 0 ];then
	collect_ipv6
fi

#/sbin/service iptables save
judge_os
if [ "$?" -eq 7 ];then
	chmod +x /etc/rc.local
	fgrep 'ipt_ser_traffic.sh' /etc/rc.local -q || echo 'bash /usr/local/bin/ipt_ser_traffic.sh &> /dev/null' >> /etc/rc.local
else
	/sbin/service iptables save		
fi
exit 0



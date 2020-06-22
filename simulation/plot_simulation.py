import matplotlib.pyplot as plt
import pandas as pd
import os
import numpy as np 

data_dir = './test_data/'
files = [(data_dir + fname) for fname in os.listdir(data_dir)\
         if fname.startswith('service') and fname.endswith('.csv')]

def read_all_files(files):
    df = pd.DataFrame()
    for fname in files:
        data = pd.read_csv(fname)
        print(data['hosts'])
        print(data['keep'])
        print(data['prepare_wall_max'])
        print(data['authorize_wall_max'])
        print(data['sign_deferred_wall_avg'])
        print(data['execute_deferred_wall_avg'])
        print(data['reject_wall_avg'])
        print(data['round_wall_avg'])
    return data


df = read_all_files(files)
keep = list(set(df['keep']))
hosts = list(set(df['hosts']))
queries = list(set(df['queries']))
rounds = list(set(df['rounds']))


for k in keep:
        titlestring = 'MedChain time measurements for 1 query'
        # No whitespace, colons or commata in filenames
        namestring = titlestring.replace(' ','').replace(':','-').replace(',','_')
        data = df.loc[df['keep'] == k].sort_values('hosts')
        data = data.reset_index()
    

        ax0 = data.plot(y='round_wall_avg', marker='o')                
        ax = data.plot(kind='bar',\
                x='hosts',\
                y=['prepare_wall_avg','sign_deferred_wall_avg','execute_deferred_wall_avg','reject_wall_avg','authorize_wall_avg'],\
                stacked=True,ax=ax0)
        
        plt.xlabel('Number of MedChain Nodes')
        plt.ylabel('Time (second)')
        plt.title(titlestring)
        plt.savefig(data_dir + 'barplot' + '.png')
        plt.close()


        ax = data.plot(kind='bar',\
                x='hosts',\
                y=['prepare_wall_avg','sign_deferred_wall_avg','execute_deferred_wall_avg','reject_wall_avg','authorize_wall_avg','round_wall_avg'],\
                stacked=False,logy=True).legend(loc='center left',bbox_to_anchor=(1.0, 0.5))
        plt.subplots_adjust(right=0.6)
        plt.xlabel('Number of MedChain Nodes')
        plt.ylabel('Logarithm of Time (seconds)')
        plt.title(titlestring)
        plt.savefig(data_dir + 'barplot_log' + '.png')
        plt.close()


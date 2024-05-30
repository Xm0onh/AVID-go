import matplotlib.pyplot as plt

# Data for Reed-Solomon (RS) codes
rs_k = [1, 5, 10, 15, 20]
rs_time = [37.20597175, 7.833354625, 4.736689834, 4.46823025, 4.241938542]

# Data for Luby Transform (LT) codes
lt_k = [1, 5, 10, 15, 20]
lt_time = [37.20597175, 8.351990791, 5.752930583, 11.341133459, 18.1427595]

# Plotting the data
plt.figure(figsize=(10, 6))
plt.plot(rs_k, rs_time, marker='o', linestyle='-', color='b', label='RS Codes')
plt.plot(lt_k, lt_time, marker='o', linestyle='-', color='r', label='LT Codes')
plt.xlabel('K')
plt.ylabel('Time (s)')
plt.title('File downloading in a distributed network with RS and LT Codes')
plt.legend()
plt.grid(True)
plt.annotate('File size: 1,000,000 ETH transactions (~10K Blocks)', xy=(0.5, 0.95), xycoords='axes fraction', fontsize=12, ha='center')

plt.show()

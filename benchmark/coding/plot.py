import matplotlib.pyplot as plt
import numpy as np

# Data for Reed-Solomon (RS) codes
rs_data_shards = [1, 5, 10, 15, 20]
rs_encode_time = [32.0, 18.0, 7.0, 10.0, 9.0]
rs_decode_time = [69.0, 47.0, 115.0, 47.0, 27.0]
rs_overhead = [2.0, 1.4, 1.3, 1.27, 1.25]

# Data for Luby Transform (LT) codes
lt_encoded_block_count = [1, 5, 10, 15, 20]
lt_encode_time = [91.0, 333.0, 475.0, 1064.0, 850.0]
lt_decode_time = [33.0, 209.0, 482.0, 619.0, 667.0]
lt_overhead = [1.0, 2.5, 3.33, 3.75, 4.0]

# Create subplots
fig, ax = plt.subplots(2, 2, figsize=(14, 10))

# Plot encoding time
ax[0, 0].plot(rs_data_shards, rs_encode_time, marker='o', linestyle='-', color='b', label='RS Encode Time')
ax[0, 0].plot(lt_encoded_block_count, lt_encode_time, marker='o', linestyle='-', color='r', label='LT Encode Time')
ax[0, 0].set_xlabel('Data Shards / Encoded Block Count')
ax[0, 0].set_ylabel('Encode Time (s)')
ax[0, 0].set_title('Encoding Time Comparison')
ax[0, 0].legend()
ax[0, 0].grid(True)

# Plot decoding time
ax[0, 1].plot(rs_data_shards, rs_decode_time, marker='o', linestyle='-', color='b', label='RS Decode Time')
ax[0, 1].plot(lt_encoded_block_count, lt_decode_time, marker='o', linestyle='-', color='r', label='LT Decode Time')
ax[0, 1].set_xlabel('Data Shards / Encoded Block Count')
ax[0, 1].set_ylabel('Decode Time (s)')
ax[0, 1].set_title('Decoding Time Comparison')
ax[0, 1].legend()
ax[0, 1].grid(True)

# Plot overhead
ax[1, 0].plot(rs_data_shards, rs_overhead, marker='o', linestyle='-', color='b', label='RS Overhead')
ax[1, 0].plot(lt_encoded_block_count, lt_overhead, marker='o', linestyle='-', color='r', label='LT Overhead')
ax[1, 0].set_xlabel('Data Shards / Encoded Block Count')
ax[1, 0].set_ylabel('Overhead')
ax[1, 0].set_title('Overhead Comparison')
ax[1, 0].legend()
ax[1, 0].grid(True)

# Hide the fourth subplot
ax[1, 1].axis('off')

# Adjust layout
plt.tight_layout()
plt.show()

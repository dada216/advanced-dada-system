import sqlite3
import sys
import glob
import os

db_files = glob.glob("/home/dada/.local/share/ads/sessions/*.db")
db_files.sort(key=os.path.getmtime, reverse=True)

for db in db_files[:1]:
    print(f"--- DB: {db} ---")
    conn = sqlite3.connect(db)
    cursor = conn.cursor()
    
    # Check command_history
    cursor.execute("SELECT id, command_text FROM command_history ORDER BY id DESC LIMIT 10")
    rows = cursor.fetchall()
    print("\nLast 10 Commands in command_history:")
    for row in rows:
        print(f"  {row[0]} | {repr(row[1])}")
        
    # Check io_stream (if we can read text)
    try:
        cursor.execute("SELECT id, length(data) FROM io_stream ORDER BY id DESC LIMIT 5")
        print("\nLast 5 io_stream chunks (id, len):")
        for row in cursor.fetchall():
            print(f"  {row[0]} | {row[1]}")
    except sqlite3.OperationalError as e:
        print(f"io_stream error: {e}")

    # Check fts_index
    try:
        cursor.execute("SELECT rowid, length(text) FROM fts_index ORDER BY rowid DESC LIMIT 5")
        print("\nLast 5 fts_index records (rowid, len):")
        for row in cursor.fetchall():
            print(f"  {row[0]} | {row[1]}")
    except sqlite3.OperationalError as e:
        print(f"fts_index error: {e}")


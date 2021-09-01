import time
import random
import math
import threading
import socket

conn = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
dest = ("127.0.0.1", 8125)

m = threading.Lock()

def send(name, value, type, *tags):
    m.acquire()
    tags = ('|#' + ','.join(tags)) if tags else ''
    msg = f"{name}:{value}|{type}{tags}"
    conn.sendto(msg.encode('utf-8'), dest)
    now = time.asctime(time.gmtime(time.time()))
    print('{}: {}'.format(now, msg))
    m.release()

def spiky_counter():
    count = 0
    while True:
        inc = random.randint(0, 20)
        if inc > 5:
            count += inc * 10
            #send("dustin.ac1062.page_views.total", count, 'c')
            send("dustin.ac1062.page_views.inc", inc * 10, 'c')
        time.sleep(1)

def smooth_counter(name, period):
    while True:
        t = time.time() * math.pi / period
        send("dustin.ac1062.smooth_count", math.sin(t) + t, 'c', f'counter_name:{name}')
        time.sleep(3)

def wave_gauge(func_name, func):
    while True:
        send("dustin.ac1062.wave", func(time.time()), 'g', f'func_name:{func_name}')
        time.sleep(1)

def spawn(fn, *args):
    def work():
        try:
            fn(*args)
        except Exception as e:
            print(e)
    t = threading.Thread(target=work)
    t.daemon = True
    t.start()

def main():
    #spawn(wave_gauge, 'sin', lambda t : math.sin(t / 100))
    #spawn(wave_gauge, 'cos', lambda t : math.cos(t / 100))
    #spawn(wave_gauge, 'tan', lambda t : math.tan(t / 100))
    spawn(spiky_counter)
    #spawn(smooth_counter, 'min', 60.0)
    #spawn(smooth_counter, '2min', 120.0)
    #spawn(smooth_counter, '5min', 300.0)

    while True:
        time.sleep(1)

main()

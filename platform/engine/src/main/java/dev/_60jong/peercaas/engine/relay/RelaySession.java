package dev._60jong.peercaas.engine.relay;

import java.net.Socket;
import java.time.Instant;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;

public class RelaySession {

    private final String token;
    private final String portKey;
    private final long createdAt;

    private final AtomicReference<Socket> firstSocket = new AtomicReference<>();
    private volatile Socket secondSocket;
    private final CountDownLatch peerLatch = new CountDownLatch(1);

    public RelaySession(String token, String portKey) {
        this.token = token;
        this.portKey = portKey;
        this.createdAt = Instant.now().getEpochSecond();
    }

    public String getToken() { return token; }
    public String getPortKey() { return portKey; }
    public long getCreatedAt() { return createdAt; }

    /**
     * 소켓을 세션에 등록한다.
     * @return true = 첫 번째 연결 (상대방 대기 필요)
     *         false = 두 번째 연결 (브릿지는 첫 번째 스레드가 담당)
     */
    public boolean registerSocket(Socket socket) {
        if (firstSocket.compareAndSet(null, socket)) {
            return true; // 첫 번째
        }
        secondSocket = socket;
        peerLatch.countDown(); // 첫 번째 스레드 깨움
        return false; // 두 번째
    }

    /**
     * 첫 번째 연결의 스레드에서 호출.
     * 두 번째 소켓이 연결될 때까지 최대 timeoutMs 대기.
     */
    public Socket awaitPeer(long timeoutMs) throws InterruptedException {
        peerLatch.await(timeoutMs, TimeUnit.MILLISECONDS);
        return secondSocket; // timeout이면 null
    }

    public boolean isExpired(long ttlSeconds) {
        return Instant.now().getEpochSecond() - createdAt > ttlSeconds;
    }
}

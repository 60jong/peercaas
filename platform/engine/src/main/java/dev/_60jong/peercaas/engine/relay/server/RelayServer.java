package dev._60jong.peercaas.engine.relay.server;

import dev._60jong.peercaas.engine.relay.RelaySession;
import dev._60jong.peercaas.engine.relay.RelaySessionStore;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.ServerSocket;
import java.net.Socket;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

@Slf4j
@RequiredArgsConstructor
@Component
public class RelayServer {

    private final RelaySessionStore sessionStore;

    @Value("${relay.port}")
    private int port;

    private ServerSocket serverSocket;
    private final ExecutorService threadPool = Executors.newCachedThreadPool();

    @PostConstruct
    public void start() throws IOException {
        serverSocket = new ServerSocket(port);
        log.info("[RelayServer] TCP relay server started on port {}", port);
        threadPool.submit(this::acceptLoop);
    }

    private void acceptLoop() {
        while (!serverSocket.isClosed()) {
            try {
                Socket socket = serverSocket.accept();
                threadPool.submit(() -> handleSocket(socket));
            } catch (IOException e) {
                if (!serverSocket.isClosed()) {
                    log.error("[RelayServer] Accept error: {}", e.getMessage());
                }
            }
        }
    }

    private void handleSocket(Socket socket) {
        String token = null;
        try {
            // 핸드셰이크: 첫 줄 = 세션 토큰
            token = readLine(socket.getInputStream());
            if (token == null || token.isBlank()) {
                log.warn("[RelayServer] Empty token, closing");
                socket.close();
                return;
            }

            RelaySession session = sessionStore.get(token).orElse(null);
            if (session == null) {
                log.warn("[RelayServer] Unknown token: {}", token);
                socket.close();
                return;
            }

            log.info("[RelayServer] Socket registered for token: {}", token);
            boolean isFirst = session.registerSocket(socket);

            if (isFirst) {
                // 첫 번째 연결: 상대방(두 번째)이 올 때까지 대기 (최대 60초)
                Socket peer = session.awaitPeer(60_000);
                if (peer == null) {
                    log.warn("[RelayServer] Peer timeout for token: {}", token);
                    socket.close();
                    sessionStore.remove(token);
                    return;
                }
                log.info("[RelayServer] Bridging session: {}", token);
                bridgeSockets(socket, peer, token);
            }
            // 두 번째 연결: 브릿지는 첫 번째 스레드가 담당, 여기서 종료

        } catch (IOException | InterruptedException e) {
            log.error("[RelayServer] Error handling socket: {}", e.getMessage());
            closeQuietly(socket);
            if (token != null) sessionStore.remove(token);
        }
    }

    /**
     * 소켓 스트림에서 '\n' 기준으로 한 줄 읽기.
     * BufferedReader를 쓰지 않는 이유: 이후 raw 바이트 복사 시 버퍼가 데이터를 삼키는 걸 방지.
     */
    private String readLine(InputStream in) throws IOException {
        StringBuilder sb = new StringBuilder();
        int b;
        while ((b = in.read()) != -1) {
            if (b == '\n') break;
            if (b != '\r') sb.append((char) b);
        }
        return sb.isEmpty() ? null : sb.toString();
    }

    private void bridgeSockets(Socket a, Socket b, String token) {
        try {
            InputStream  aIn  = a.getInputStream();
            OutputStream aOut = a.getOutputStream();
            InputStream  bIn  = b.getInputStream();
            OutputStream bOut = b.getOutputStream();

            // a→b 방향: 별도 스레드
            threadPool.submit(() -> {
                try { aIn.transferTo(bOut); } catch (IOException ignored) {}
                closeQuietly(a);
                closeQuietly(b);
            });

            // b→a 방향: 현재 스레드
            try { bIn.transferTo(aOut); } catch (IOException ignored) {}
            closeQuietly(b);
            closeQuietly(a);

        } catch (IOException e) {
            log.error("[RelayServer] Bridge setup error for {}: {}", token, e.getMessage());
        } finally {
            sessionStore.remove(token);
            log.info("[RelayServer] Session closed: {}", token);
        }
    }

    private void closeQuietly(Socket s) {
        try { s.close(); } catch (IOException ignored) {}
    }

    @PreDestroy
    public void stop() {
        log.info("[RelayServer] Shutting down");
        try { serverSocket.close(); } catch (IOException ignored) {}
        threadPool.shutdown();
    }
}

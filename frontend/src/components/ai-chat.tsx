import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import axios from "axios";
import { useSearchParams } from "react-router";
import { isRepositoryName, readDefaultRepository } from "@/lib/preferences";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import type { AuthState } from "@/hooks/use-session";
import { RepositoryPicker } from "@/components/repository-picker";
import { AnalysisResult } from "@/components/analysis-result";
import { requestError, type AIResponse } from "@/lib/analysis";
import { responseWordCount, revealResponse } from "@/lib/chat-reveal";
import { Textarea } from "@/components/ui/textarea";

type Message = {
  role: "user" | "assistant";
  content: string | AIResponse;
};

function ChatReply({ response, animate, onComplete }: {
  response: AIResponse;
  animate: boolean;
  onComplete: () => void;
}) {
  const [visibleWords, setVisibleWords] = useState(0);

  useEffect(() => {
    if (!animate) return;
    const preference = window.matchMedia("(prefers-reduced-motion: reduce)");
    if (preference.matches) {
      onComplete();
      return;
    }
    const total = responseWordCount(response);
    const start = performance.now();
    // Keep long replies from spending more than six seconds in the reveal.
    const wordsPerMs = Math.max(0.05, total / 6000);
    const timer = window.setInterval(() => {
      const count = Math.min(total, Math.ceil((performance.now() - start) * wordsPerMs));
      setVisibleWords(count);
      if (count >= total) onComplete();
    }, 40);
    const reduceMotion = () => { if (preference.matches) onComplete(); };
    preference.addEventListener("change", reduceMotion);
    return () => {
      window.clearInterval(timer);
      preference.removeEventListener("change", reduceMotion);
    };
  }, [response, animate, onComplete]);

  return (
    <article aria-label="devy response" className="space-y-3">
      <div className="flex min-h-11 flex-wrap items-center justify-between gap-2">
        <p className="font-serif text-base italic">devy</p>
        {animate && <Button type="button" variant="ghost" onClick={onComplete}>Show full response</Button>}
      </div>
      <AnalysisResult response={animate ? revealResponse(response, visibleWords) : response} />
    </article>
  );
}

export function AIChat({ auth }: { auth: AuthState }) {
  const [params] = useSearchParams();
  const [repository, setRepository] = useState(() => params.get("repository") ?? readDefaultRepository());
  const [question, setQuestion] = useState(() => params.get("question") ?? "");
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(false);
  const [typing, setTyping] = useState(false);
  const busy = loading || typing;
  const finishReply = useCallback(() => setTyping(false), []);

  async function sendMessage() {
    if (!repository.trim() || !question.trim() || busy) return;

    if (!isRepositoryName(repository.trim())) {
      toast.error("Use the format owner/repository.");
      return;
    }
    const currentQuestion = question.trim();

    setLoading(true);

    try {
      const { data } = await axios.post<AIResponse>(
        "/api/chat",
        { repository: repository.trim(), question: currentQuestion },
        { timeout: 75_000 },
      );

      setTyping(true);
      setMessages((prev) => [
        ...prev,
        { role: "user", content: currentQuestion },
        { role: "assistant", content: data },
      ]);
      setQuestion("");
    } catch (error) {
      toast.error(requestError(error));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mx-auto flex h-full w-full max-w-3xl flex-1 flex-col gap-6">
      <div>
        <h1 className="text-4xl leading-none font-normal tracking-[-0.04em] md:text-5xl">
          Chat with devy
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Ask devy about your project using its GitHub file names and README as context.
        </p>
      </div>

      <RepositoryPicker user={auth.user} value={repository} onChange={(value) => { setRepository(value); setMessages([]); }} disabled={busy} />

      <div
        role="log"
        aria-label="Chat messages"
        aria-busy={busy}
        className="flex min-h-48 flex-1 flex-col gap-4 overflow-y-auto"
      >
        {messages.length === 0 && !loading && (
          <div className="flex flex-1 items-center justify-center text-sm text-muted-foreground">
            Ask your first question to get started.
          </div>
        )}

        {messages.map((message, index) => (
          <Card
            key={index}
            className={
              message.role === "user"
                ? "ml-auto max-w-[85%] bg-muted"
                : "mr-auto w-full"
            }
          >
            <CardContent className="space-y-3 text-sm wrap-anywhere">
              {typeof message.content === "string" ? (
                <p>{message.content}</p>
              ) : (
                <ChatReply response={message.content} animate={typing && index === messages.length - 1} onComplete={finishReply} />
              )}
            </CardContent>
          </Card>
        ))}

        {loading && (
          <p role="status" className="text-sm text-muted-foreground">
            devy is reading repository context and thinking…
          </p>
        )}
      </div>

      <form className="space-y-3 border-t pt-4" onSubmit={(event) => { event.preventDefault(); void sendMessage(); }}>
        <label htmlFor="question" className="text-sm font-medium">
          Your question
        </label>
        <Textarea
          id="question"
          placeholder="Ask devy about this project…"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.shiftKey && !event.nativeEvent.isComposing) {
              event.preventDefault();
              if (!event.repeat) event.currentTarget.form?.requestSubmit();
            }
          }}
          disabled={busy}
          aria-describedby="question-hint"
          rows={3}
        />

        <div className="flex flex-wrap items-center justify-between gap-3">
          <p id="question-hint" className="text-xs text-muted-foreground">Enter to send · Shift + Enter for a new line</p>
          <Button
            type="submit"
            disabled={!repository.trim() || !question.trim() || busy}
          >
            {loading ? "Thinking…" : typing ? "Writing…" : "Send message"}
          </Button>
        </div>
      </form>
    </div>
  );
}

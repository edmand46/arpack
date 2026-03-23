using System;
using System.Diagnostics;
using Google.Protobuf;
using UnityEngine;

public class BenchmarkRunner : MonoBehaviour
{
    private const int N = 10_000;
    private const int Warmup = 1_000;

    private unsafe void Start()
    {
        var apMsg = new Arpack.Messages.MoveMessage
        {
            Position  = new Arpack.Messages.Vector3 { X = 100, Y = -50, Z = 0 },
            Velocity  = new float[] { 1.5f, -2.5f, 0f },
            Waypoints = new Arpack.Messages.Vector3[]
            {
                new Arpack.Messages.Vector3 { X = 10,  Y = 20, Z = 0   },
                new Arpack.Messages.Vector3 { X = -10, Y = 0,  Z = 100 },
            },
            PlayerID = 999,
            Active   = true,
            Visible  = false,
            Ghost    = true,
            Name     = "PlayerOne",
        };

        var pbMsg = new Benchproto.MoveMessage
        {
            Position = new Benchproto.Vector3 { X = 100, Y = -50, Z = 0 },
            PlayerId = 999,
            Active   = true,
            Visible  = false,
            Ghost    = true,
            Name     = "PlayerOne",
        };
        pbMsg.Velocity.AddRange(new float[] { 1.5f, -2.5f, 0f });
        pbMsg.Waypoints.Add(new Benchproto.Vector3 { X = 10,  Y = 20, Z = 0   });
        pbMsg.Waypoints.Add(new Benchproto.Vector3 { X = -10, Y = 0,  Z = 100 });

        byte[] apBuf = new byte[256];
        int apWireSize;
        fixed (byte* ptr = apBuf) { apWireSize = apMsg.Serialize(ptr); }

        byte[] apBytes = new byte[apWireSize];
        Array.Copy(apBuf, apBytes, apWireSize);

        byte[] pbBytes = pbMsg.ToByteArray();
        int pbWireSize = pbBytes.Length;
        byte[] protoOutputBuf = new byte[256];

        // Warmup (JIT)
        for (int i = 0; i < Warmup; i++)
        {
            fixed (byte* ptr = apBuf) { apMsg.Serialize(ptr); }
            fixed (byte* ptr = apBytes) { Arpack.Messages.MoveMessage.Deserialize(ptr, out _); }
            _ = pbMsg.ToByteArray();
            _ = Benchproto.MoveMessage.Parser.ParseFrom(pbBytes);
            var cos = new CodedOutputStream(protoOutputBuf);
            pbMsg.WriteTo(cos);
            cos.Flush();
        }

        Stopwatch sw;
        long gcBefore, gcAfter;

        // ArPack Serialize
        GC.Collect(); GC.WaitForPendingFinalizers(); GC.Collect();
        gcBefore = GC.GetTotalMemory(false);
        sw = Stopwatch.StartNew();
        for (int i = 0; i < N; i++)
        {
            fixed (byte* ptr = apBuf) { apMsg.Serialize(ptr); }
        }
        sw.Stop();
        gcAfter = GC.GetTotalMemory(false);
        Log("ArPack Serialize        ", sw, N, gcAfter - gcBefore);

        // ArPack Deserialize
        GC.Collect(); GC.WaitForPendingFinalizers(); GC.Collect();
        gcBefore = GC.GetTotalMemory(false);
        sw = Stopwatch.StartNew();
        for (int i = 0; i < N; i++)
        {
            fixed (byte* ptr = apBytes) { Arpack.Messages.MoveMessage.Deserialize(ptr, out _); }
        }
        sw.Stop();
        gcAfter = GC.GetTotalMemory(false);
        Log("ArPack Deserialize      ", sw, N, gcAfter - gcBefore);

        // Proto Serialize (alloc)
        GC.Collect(); GC.WaitForPendingFinalizers(); GC.Collect();
        gcBefore = GC.GetTotalMemory(false);
        sw = Stopwatch.StartNew();
        byte[] pbOut = null;
        for (int i = 0; i < N; i++)
        {
            pbOut = pbMsg.ToByteArray();
        }
        sw.Stop();
        gcAfter = GC.GetTotalMemory(false);
        Log("Proto Serialize (alloc) ", sw, N, gcAfter - gcBefore);
        _ = pbOut;

        // Proto Deserialize (alloc)
        GC.Collect(); GC.WaitForPendingFinalizers(); GC.Collect();
        gcBefore = GC.GetTotalMemory(false);
        sw = Stopwatch.StartNew();
        for (int i = 0; i < N; i++)
        {
            _ = Benchproto.MoveMessage.Parser.ParseFrom(pbBytes);
        }
        sw.Stop();
        gcAfter = GC.GetTotalMemory(false);
        Log("Proto Deserialize (alloc)", sw, N, gcAfter - gcBefore);

        // Proto Serialize (reuse buffer)
        GC.Collect(); GC.WaitForPendingFinalizers(); GC.Collect();
        gcBefore = GC.GetTotalMemory(false);
        sw = Stopwatch.StartNew();
        for (int i = 0; i < N; i++)
        {
            var cos = new CodedOutputStream(protoOutputBuf);
            pbMsg.WriteTo(cos);
            cos.Flush();
        }
        sw.Stop();
        gcAfter = GC.GetTotalMemory(false);
        Log("Proto Serialize (reuse) ", sw, N, gcAfter - gcBefore);

        UnityEngine.Debug.Log($"[Bench] Wire sizes — ArPack: {apWireSize} bytes | Protobuf: {pbWireSize} bytes");
    }

    private static void Log(string label, Stopwatch sw, int n, long gcDelta)
    {
        double nsPerOp = sw.Elapsed.TotalMilliseconds * 1_000_000.0 / n;
        long bPerOp = Math.Max(0, gcDelta) / n;
        UnityEngine.Debug.Log($"[Bench] {label}: {nsPerOp,8:F1} ns/op | {bPerOp,6} B/op");
    }
}
